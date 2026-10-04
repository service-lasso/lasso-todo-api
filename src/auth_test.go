package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResourceServerDeniesBeforeMutation(t *testing.T) {
	var issuer string
	mode := "valid"
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/v2/introspect" || r.Method != "POST" {
			t.Error("unexpected introspection request")
			w.WriteHeader(400)
			return
		}
		id, secret, ok := r.BasicAuth()
		if !ok || id != "api-client" || secret != "fixture-secret" {
			t.Error("missing API credential")
			w.WriteHeader(401)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("token_type_hint") != "access_token" {
			t.Error("wrong token type hint")
		}
		if mode == "outage" {
			w.WriteHeader(503)
			return
		}
		if mode == "redirect" {
			w.Header().Set("Location", "https://other.invalid/")
			w.WriteHeader(302)
			return
		}
		if mode == "malformed" {
			_, _ = w.Write([]byte("invalid"))
			return
		}
		if mode == "oversize" {
			_, _ = w.Write([]byte(strings.Repeat(" ", 65537)))
			return
		}
		response := map[string]any{"active": true, "iss": issuer, "aud": []string{"project"}, "client_id": "todo-client", "sub": "user", "token_type": "Bearer", "scope": "openid profile", "exp": time.Now().Unix() + 600}
		if r.Form.Get("token") != "user-access" {
			response["active"] = false
		}
		switch mode {
		case "revoked":
			response["active"] = false
		case "expired":
			response["exp"] = time.Now().Unix() - 1
		case "future":
			response["nbf"] = time.Now().Unix() + 60
		case "issuer":
			response["iss"] = "https://other.invalid"
		case "audience":
			response["aud"] = []string{"another-project"}
		case "client":
			response["client_id"] = "another-client"
		case "subject":
			response["sub"] = ""
		case "type":
			response["token_type"] = "ID"
		case "scope":
			response["scope"] = "profile"
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer provider.Close()
	issuer = provider.URL
	client := provider.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.Timeout = time.Second
	auth := &apiAuth{issuer: issuer, audience: "project", todoClient: "todo-client", clientID: "api-client", secret: "fixture-secret", client: client}
	var reads, writes atomic.Int32
	api := httptest.NewServer(auth.protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			writes.Add(1)
			w.WriteHeader(201)
		} else {
			reads.Add(1)
			w.WriteHeader(200)
		}
	})))
	defer api.Close()
	call := func(method, header string, duplicate bool) int {
		req, _ := http.NewRequest(method, api.URL+"/todos", strings.NewReader(`{"title":"blocked"}`))
		if header != "" {
			req.Header.Add("Authorization", header)
		}
		if duplicate {
			req.Header.Add("Authorization", "Bearer user-access")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal("request failed")
		}
		defer res.Body.Close()
		return res.StatusCode
	}
	for _, entry := range []struct {
		name, header string
		duplicate    bool
	}{{"missing", "", false}, {"wrong", "Bearer wrong-access", false}, {"ID token", "Bearer id-token", false}, {"scheme", "Basic user-access", false}, {"spaces", "Bearer  user-access", false}, {"duplicate", "Bearer user-access", true}, {"oversized", "Bearer " + strings.Repeat("x", 8192), false}} {
		t.Run(entry.name, func(t *testing.T) {
			if call("GET", entry.header, entry.duplicate) != 401 || call("POST", entry.header, entry.duplicate) != 401 {
				t.Fatal("unauthenticated access accepted")
			}
		})
	}
	if reads.Load() != 0 || writes.Load() != 0 {
		t.Fatal("unauthenticated handler/database reached")
	}
	if call("GET", "Bearer user-access", false) != 200 || call("POST", "Bearer user-access", false) != 201 {
		t.Fatal("valid access token rejected")
	}
	for _, rejected := range []string{"revoked", "expired", "future", "issuer", "audience", "client", "subject", "type", "scope", "outage", "redirect", "malformed", "oversize"} {
		t.Run(rejected, func(t *testing.T) {
			mode = rejected
			expected := 401
			if rejected == "outage" || rejected == "redirect" || rejected == "malformed" || rejected == "oversize" {
				expected = 503
			}
			if call("GET", "Bearer user-access", false) != expected || call("POST", "Bearer user-access", false) != expected {
				t.Fatal("invalid token/provider response accepted")
			}
		})
	}
	if reads.Load() != 1 || writes.Load() != 1 {
		t.Fatal("rejected request mutated/read protected handler")
	}
	res, err := http.Get(api.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("health must remain public")
	}
}

func TestExplicitAuthConfiguration(t *testing.T) {
	for _, key := range []string{"TODO_API_AUTH_MODE", "TODO_OIDC_ISSUER", "TODO_OIDC_AUDIENCE", "TODO_OIDC_CLIENT_ID", "TODO_API_CLIENT_ID", "TODO_API_CLIENT_SECRET_FILE", "TODO_API_CA_FILE"} {
		t.Setenv(key, "")
	}
	if _, err := authFromEnvironment(); err == nil {
		t.Fatal("missing mode silently permits access")
	}
	t.Setenv("TODO_API_AUTH_MODE", "anonymous")
	if auth, err := authFromEnvironment(); err != nil || !auth.anonymous {
		t.Fatal("explicit earlier lesson mode rejected")
	}
	t.Setenv("TODO_OIDC_ISSUER", "https://identity.invalid")
	if _, err := authFromEnvironment(); err == nil {
		t.Fatal("partial identity falls back to anonymous")
	}
	t.Setenv("TODO_API_AUTH_MODE", "zitadel")
	if _, err := authFromEnvironment(); err == nil {
		t.Fatal("partial identity accepted")
	}
	secret := filepath.Join(t.TempDir(), "api-secret")
	if err := os.WriteFile(secret, []byte("fixture-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TODO_OIDC_AUDIENCE", "project")
	t.Setenv("TODO_OIDC_CLIENT_ID", "todo-client")
	t.Setenv("TODO_API_CLIENT_ID", "api-client")
	t.Setenv("TODO_API_CLIENT_SECRET_FILE", secret)
	if _, err := authFromEnvironment(); err != nil {
		t.Fatal("complete private configuration rejected")
	}
	t.Setenv("TODO_OIDC_ISSUER", "http://identity.invalid")
	if _, err := authFromEnvironment(); err == nil {
		t.Fatal("plaintext issuer accepted")
	}
}

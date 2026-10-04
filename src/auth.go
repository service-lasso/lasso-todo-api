package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type apiAuth struct {
	anonymous                                      bool
	issuer, audience, todoClient, clientID, secret string
	client                                         *http.Client
}

var bearerToken = regexp.MustCompile(`^[A-Za-z0-9._~+/-]+={0,2}$`)

func authFromEnvironment() (*apiAuth, error) {
	mode := os.Getenv("TODO_API_AUTH_MODE")
	keys := []string{"TODO_OIDC_ISSUER", "TODO_OIDC_AUDIENCE", "TODO_OIDC_CLIENT_ID", "TODO_API_CLIENT_ID", "TODO_API_CLIENT_SECRET_FILE", "TODO_API_CA_FILE"}
	if mode == "anonymous" {
		for _, key := range keys {
			if os.Getenv(key) != "" {
				return nil, errors.New("anonymous mode cannot contain identity configuration")
			}
		}
		return &apiAuth{anonymous: true}, nil
	}
	if mode != "zitadel" {
		return nil, errors.New("explicit API authentication mode required")
	}
	a := &apiAuth{issuer: os.Getenv(keys[0]), audience: os.Getenv(keys[1]), todoClient: os.Getenv(keys[2]), clientID: os.Getenv(keys[3])}
	issuer, err := url.Parse(a.issuer)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" || (issuer.Path != "" && issuer.Path != "/") {
		return nil, errors.New("trusted HTTPS issuer origin required")
	}
	a.issuer = strings.TrimSuffix(a.issuer, "/")
	for _, value := range []string{a.audience, a.todoClient, a.clientID} {
		if value == "" || len(value) > 256 || strings.ContainsAny(value, " \r\n\t${}") {
			return nil, errors.New("complete identity IDs required")
		}
	}
	secretPath := os.Getenv(keys[4])
	info, err := os.Stat(secretPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 8192 {
		return nil, errors.New("private API credential file required")
	}
	// Windows ACLs are provisioned by the operator; Unix files must be owner-only.
	if info.Mode().Perm()&0077 != 0 && os.PathSeparator != '\\' {
		return nil, errors.New("API credential file must be owner-only")
	}
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		return nil, errors.New("cannot read API credential")
	}
	a.secret = strings.TrimSpace(string(secret))
	if a.secret == "" || strings.ContainsAny(a.secret, "\r\n") {
		return nil, errors.New("invalid API credential")
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if ca := os.Getenv(keys[5]); ca != "" {
		bytes, err := os.ReadFile(ca)
		if err != nil || !roots.AppendCertsFromPEM(bytes) {
			return nil, errors.New("invalid explicit identity CA")
		}
	}
	a.client = &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, MaxConnsPerHost: 16, MaxIdleConnsPerHost: 4, ResponseHeaderTimeout: 3 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return a, nil
}

func (a *apiAuth) validate(ctx context.Context, token string) (bool, error) {
	form := url.Values{"token": {token}, "token_type_hint": {"access_token"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.issuer+"/oauth/v2/introspect", strings.NewReader(form.Encode()))
	if err != nil {
		return false, errors.New("identity unavailable")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(a.clientID), url.QueryEscape(a.secret))
	response, err := a.client.Do(req)
	if err != nil {
		return false, errors.New("identity unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errors.New("identity unavailable")
	}
	bytes, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(bytes) > 65536 {
		return false, errors.New("identity unavailable")
	}
	var result struct {
		Active    bool            `json:"active"`
		Issuer    string          `json:"iss"`
		Audience  json.RawMessage `json:"aud"`
		ClientID  string          `json:"client_id"`
		Subject   string          `json:"sub"`
		Type      string          `json:"token_type"`
		Scope     string          `json:"scope"`
		Expires   int64           `json:"exp"`
		NotBefore int64           `json:"nbf"`
	}
	if json.Unmarshal(bytes, &result) != nil {
		return false, errors.New("identity unavailable")
	}
	now := time.Now().Unix()
	if !result.Active || result.Issuer != a.issuer || result.ClientID != a.todoClient || result.Subject == "" || result.Type != "Bearer" || result.Expires <= now || result.NotBefore > now {
		return false, nil
	}
	var audience []string
	var single string
	if json.Unmarshal(result.Audience, &audience) != nil {
		if json.Unmarshal(result.Audience, &single) != nil {
			return false, nil
		}
		audience = []string{single}
	}
	intended := false
	for _, item := range audience {
		if item == a.audience {
			intended = true
		}
	}
	scope := false
	for _, item := range strings.Fields(result.Scope) {
		if item == "openid" {
			scope = true
		}
	}
	return intended && scope, nil
}

func (a *apiAuth) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.anonymous || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		deny := func(status int, message string) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			if status == 401 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="todo-api"`)
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		}
		headers := r.Header.Values("Authorization")
		if len(headers) != 1 || len(headers[0]) > 8192 {
			deny(401, "Access token required")
			return
		}
		parts := strings.Split(headers[0], " ")
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !bearerToken.MatchString(parts[1]) {
			deny(401, "Access token required")
			return
		}
		allowed, err := a.validate(r.Context(), parts[1])
		if err != nil {
			deny(503, "Identity validation unavailable")
			return
		}
		if !allowed {
			deny(401, "Invalid access token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

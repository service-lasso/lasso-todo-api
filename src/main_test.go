package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidInputCannotReachDatabase(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, contentType, origin string
		status                                  int
	}{
		{"malformed JSON", "POST", "{", "application/json", "", 400},
		{"empty title", "POST", `{"title":" "}`, "application/json", "", 400},
		{"long title", "POST", `{"title":"` + strings.Repeat("x", 201) + `"}`, "application/json", "", 400},
		{"foreign browser origin", "POST", `{"title":"test"}`, "application/json", "https://foreign.example", 403},
		{"wrong media type", "POST", `{"title":"test"}`, "text/plain", "", 415},
		{"unsupported method", "DELETE", "", "", "", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "/todos", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Origin", tc.origin)
			response := httptest.NewRecorder()
			todoHandler(nil).ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("got%d, want%d", response.Code, tc.status)
			}
		})
	}
}

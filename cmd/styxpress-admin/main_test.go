package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddedSPADoesNotExposeSessionBootstrap(t *testing.T) {
	handler := embeddedSPA()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "__STYXPRESS_SESSION__") {
		t.Fatalf("anonymous SPA response contains session bootstrap: %s", body)
	}
}

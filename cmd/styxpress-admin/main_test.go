package main

import (
	"strings"
	"testing"
)

func TestInjectSessionBootstrapAddsEscapedTokenBeforeHead(t *testing.T) {
	index := []byte("<!doctype html><html><head><title>Admin</title></head><body></body></html>")
	token := `abc"</script><script>alert(1)</script>`

	got := string(injectSessionBootstrap(index, token))
	want := `<script>window.__STYXPRESS_SESSION__="abc\"\u003c/script\u003e\u003cscript\u003ealert(1)\u003c/script\u003e";</script>`

	if !strings.Contains(got, want) {
		t.Fatalf("bootstrap script missing or unescaped: %s", got)
	}
	if strings.Index(got, "<script>window.__STYXPRESS_SESSION__") > strings.Index(got, "</head>") {
		t.Fatalf("bootstrap script was not injected before </head>: %s", got)
	}
	if strings.Contains(got, token) {
		t.Fatalf("raw token was injected without JSON escaping: %s", got)
	}
}

func TestInjectSessionBootstrapPrependsWhenHeadIsMissing(t *testing.T) {
	got := string(injectSessionBootstrap([]byte("<html></html>"), "local-token"))
	wantPrefix := "<script>window.__STYXPRESS_SESSION__=\"local-token\";</script>\n<html>"

	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("bootstrap script was not prepended: %s", got)
	}
}

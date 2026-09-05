package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	t.Setenv("COMMIT_SHA", "")

	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/health")
	page.AssertSuccessResponse()
	if page.Status() != http.StatusOK {
		t.Errorf("status %d, want 200", page.Status())
	}
	if got := page.Header("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
	if got := page.Header("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag %q, want noindex", got)
	}
	if ct := page.Header("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type %q, want application/json", ct)
	}

	var health healthResponse
	if err := json.Unmarshal(page.RawBody(), &health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("status %q, want ok", health.Status)
	}
	if health.Revision != defaultRevision {
		t.Errorf("revision %q, want %q", health.Revision, defaultRevision)
	}
}

func TestHealthRevisionFromEnv(t *testing.T) {
	t.Setenv("COMMIT_SHA", "abc123def")

	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/health")
	page.AssertSuccessResponse()

	var health healthResponse
	if err := json.Unmarshal(page.RawBody(), &health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("status %q, want ok", health.Status)
	}
	if health.Revision != "abc123def" {
		t.Errorf("revision %q, want abc123def", health.Revision)
	}
}

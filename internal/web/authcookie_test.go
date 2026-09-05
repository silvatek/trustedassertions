package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
)

func initTestUserJwt(t *testing.T) {
	t.Helper()
	if err := auth.InitUserJwt("test-user-jwt-key-32-bytes-long!!", time.Hour); err != nil {
		t.Fatalf("InitUserJwt: %v", err)
	}
}

func TestAuthCookieIsHttpOnly(t *testing.T) {
	initTestUserJwt(t)

	cookie := MakeAuthCookie("tester")
	if !cookie.HttpOnly {
		t.Error("Auth cookie should be HttpOnly")
	}
	if cookie.Secure {
		t.Error("Auth cookie for tests should not default to Secure")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Error("Auth cookie should use SameSite=Strict")
	}
}

func TestSetAuthCookieSecureOnHTTPS(t *testing.T) {
	initTestUserJwt(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/web/home", nil)
	req.TLS = &tls.ConnectionState{}
	if err := SetAuthCookie("tester", rec, req); err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 auth cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Error("Auth cookie should be HttpOnly")
	}
	if !cookies[0].Secure {
		t.Error("Auth cookie should be Secure on HTTPS")
	}
}

func TestSetAuthCookieSecureOnForwardedProto(t *testing.T) {
	initTestUserJwt(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://example.com/web/home", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if err := SetAuthCookie("tester", rec, req); err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	cookie := rec.Result().Cookies()[0]
	if !cookie.Secure {
		t.Error("Auth cookie should be Secure when X-Forwarded-Proto is https")
	}
	if !cookie.HttpOnly {
		t.Error("Auth cookie should be HttpOnly")
	}
}

func TestSetAuthCookieNotSecureOnHTTP(t *testing.T) {
	initTestUserJwt(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/home", nil)
	if err := SetAuthCookie("tester", rec, req); err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Secure {
		t.Error("Auth cookie should not be Secure on plain HTTP")
	}
	if !cookie.HttpOnly {
		t.Error("Auth cookie should still be HttpOnly on HTTP")
	}
}

func TestAuthCookieUsesConfiguredTTL(t *testing.T) {
	prevTTL := auth.UserJwtTTL()
	t.Cleanup(func() {
		if err := auth.InitUserJwt("test-user-jwt-key-32-bytes-long!!", prevTTL); err != nil {
			t.Errorf("restore InitUserJwt: %v", err)
		}
	})
	if err := auth.InitUserJwt("test-user-jwt-key-32-bytes-long!!", 2*time.Second); err != nil {
		t.Fatalf("InitUserJwt: %v", err)
	}

	cookie := MakeAuthCookie("tester")
	if cookie.MaxAge != 2 {
		t.Errorf("MaxAge = %d, want 2", cookie.MaxAge)
	}
	remaining := time.Until(cookie.Expires)
	if remaining < time.Second || remaining > 3*time.Second {
		t.Errorf("Expires remaining %v, want about 2s", remaining)
	}

	username, err := auth.ParseUserJwt(cookie.Value)
	if err != nil {
		t.Fatalf("cookie JWT should parse with the configured key: %v", err)
	}
	if username != "tester" {
		t.Errorf("username = %q, want tester", username)
	}
}

func TestSetAuthCookieFailsWithoutKey(t *testing.T) {
	restoreTTL := auth.UserJwtTTL()
	t.Cleanup(func() {
		if err := auth.InitUserJwt("test-user-jwt-key-32-bytes-long!!", restoreTTL); err != nil {
			t.Errorf("restore InitUserJwt: %v", err)
		}
	})
	t.Setenv("USER_JWT_KEY", "")
	t.Setenv("USER_JWT_TTL", "")
	t.Setenv("GCLOUD_PROJECT", "")
	if err := auth.InitUserJwtFromEnv(); err != nil {
		t.Fatalf("InitUserJwtFromEnv: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/login", nil)
	err := SetAuthCookie("tester", rec, req)
	if err == nil {
		t.Fatal("expected SetAuthCookie to fail when USER_JWT_KEY is unset")
	}
	if !strings.Contains(err.Error(), "USER_JWT_KEY is not set") {
		t.Errorf("error %q should mention USER_JWT_KEY is not set", err)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("should not write an auth cookie when JWT creation fails")
	}
}

func TestClearAuthCookieFlags(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/web/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if err := SetAuthCookie("", rec, req); err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Value != "" {
		t.Error("Cleared auth cookie should have empty value")
	}
	if !cookie.HttpOnly || !cookie.Secure {
		t.Error("Cleared auth cookie should keep HttpOnly and Secure so the browser will delete it")
	}
}

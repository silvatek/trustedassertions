package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
)

func TestAuthCookieIsHttpOnly(t *testing.T) {
	userJwtKey = auth.MakeJwtKey()

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
	userJwtKey = auth.MakeJwtKey()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/web/home", nil)
	req.TLS = &tls.ConnectionState{}
	SetAuthCookie("tester", rec, req)

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
	userJwtKey = auth.MakeJwtKey()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://example.com/web/home", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	SetAuthCookie("tester", rec, req)

	cookie := rec.Result().Cookies()[0]
	if !cookie.Secure {
		t.Error("Auth cookie should be Secure when X-Forwarded-Proto is https")
	}
	if !cookie.HttpOnly {
		t.Error("Auth cookie should be HttpOnly")
	}
}

func TestSetAuthCookieNotSecureOnHTTP(t *testing.T) {
	userJwtKey = auth.MakeJwtKey()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/home", nil)
	SetAuthCookie("tester", rec, req)

	cookie := rec.Result().Cookies()[0]
	if cookie.Secure {
		t.Error("Auth cookie should not be Secure on plain HTTP")
	}
	if !cookie.HttpOnly {
		t.Error("Auth cookie should still be HttpOnly on HTTP")
	}
}

func TestClearAuthCookieFlags(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/web/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	SetAuthCookie("", rec, req)

	cookie := rec.Result().Cookies()[0]
	if cookie.Value != "" {
		t.Error("Cleared auth cookie should have empty value")
	}
	if !cookie.HttpOnly || !cookie.Secure {
		t.Error("Cleared auth cookie should keep HttpOnly and Secure so the browser will delete it")
	}
}

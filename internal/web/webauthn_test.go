package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestInitWebAuthn(t *testing.T) {
	if err := InitWebAuthn("http://127.0.0.1:8080", "Trusted Assertions", "default_csrf_key"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}
	if WebAuthn() == nil {
		t.Fatal("WebAuthn instance was not stored")
	}
	if WebAuthn().Config.RPID != "127.0.0.1" {
		t.Errorf("RPID = %q", WebAuthn().Config.RPID)
	}
	if WebAuthn().Config.RPDisplayName != "Trusted Assertions" {
		t.Errorf("RPDisplayName = %q", WebAuthn().Config.RPDisplayName)
	}
}

func TestInitWebAuthnDerivesRPIDFromOrigin(t *testing.T) {
	if err := InitWebAuthn("https://trustedassertions.silvatek.uk/web/home", "Trusted Assertions", "secret"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}
	if WebAuthn().Config.RPID != "trustedassertions.silvatek.uk" {
		t.Errorf("RPID = %q", WebAuthn().Config.RPID)
	}
	origins := WebAuthn().Config.RPOrigins
	if len(origins) != 1 || origins[0] != "https://trustedassertions.silvatek.uk" {
		t.Errorf("RPOrigins = %v", origins)
	}
}

func TestInitWebAuthnRequiresOrigin(t *testing.T) {
	if err := InitWebAuthn("", "Trusted Assertions", "secret"); err == nil {
		t.Error("expected error when RP origin is empty")
	}
	if err := InitWebAuthn("trustedassertions.silvatek.uk", "Trusted Assertions", "secret"); err == nil {
		t.Error("expected error when RP origin has no scheme")
	}
}

func TestWebAuthnSessionRoundTrip(t *testing.T) {
	if err := InitWebAuthn("http://127.0.0.1:8080", "Trusted Assertions", "default_csrf_key"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}

	session := &webauthn.SessionData{
		Challenge:      "abc123",
		RelyingPartyID: "127.0.0.1",
		UserID:         []byte("alice"),
		Expires:        time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/passkey", nil)
	if err := SetWebAuthnSession(session, rec, req); err != nil {
		t.Fatalf("SetWebAuthnSession: %v", err)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != webAuthnSessionCookie {
		t.Errorf("cookie name = %q", cookie.Name)
	}
	if !cookie.HttpOnly {
		t.Error("ceremony cookie should be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Error("ceremony cookie should use SameSite=Strict")
	}
	if cookie.Secure {
		t.Error("ceremony cookie should not be Secure on HTTP")
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path = %q", cookie.Path)
	}

	readReq := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/passkey", nil)
	readReq.AddCookie(cookie)
	got, err := ReadWebAuthnSession(readReq)
	if err != nil {
		t.Fatalf("ReadWebAuthnSession: %v", err)
	}
	if got.Challenge != session.Challenge {
		t.Errorf("challenge = %q, want %q", got.Challenge, session.Challenge)
	}
	if got.RelyingPartyID != session.RelyingPartyID {
		t.Errorf("rpId = %q, want %q", got.RelyingPartyID, session.RelyingPartyID)
	}
	if string(got.UserID) != "alice" {
		t.Errorf("user id = %q", got.UserID)
	}
}

func TestWebAuthnSessionSecureOnHTTPS(t *testing.T) {
	if err := InitWebAuthn("https://example.com", "Trusted Assertions", "secret"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/web/passkey", nil)
	req.TLS = &tls.ConnectionState{}
	session := &webauthn.SessionData{Challenge: "x", Expires: time.Now().Add(time.Minute)}
	if err := SetWebAuthnSession(session, rec, req); err != nil {
		t.Fatalf("SetWebAuthnSession: %v", err)
	}
	if !rec.Result().Cookies()[0].Secure {
		t.Error("ceremony cookie should be Secure on HTTPS")
	}
}

func TestWebAuthnSessionRejectsTampering(t *testing.T) {
	if err := InitWebAuthn("http://127.0.0.1:8080", "Trusted Assertions", "secret"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	if err := SetWebAuthnSession(&webauthn.SessionData{Challenge: "x", Expires: time.Now().Add(time.Minute)}, rec, req); err != nil {
		t.Fatalf("SetWebAuthnSession: %v", err)
	}

	cookie := rec.Result().Cookies()[0]
	cookie.Value = cookie.Value + "tamper"
	readReq := httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	readReq.AddCookie(cookie)
	if _, err := ReadWebAuthnSession(readReq); err == nil {
		t.Error("expected error for tampered ceremony cookie")
	}
}

func TestClearWebAuthnSession(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	ClearWebAuthnSession(rec, req)

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != webAuthnSessionCookie {
		t.Errorf("cookie name = %q", cookie.Name)
	}
	if cookie.Value != "" {
		t.Error("cleared ceremony cookie should have empty value")
	}
	if cookie.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1", cookie.MaxAge)
	}
	if !cookie.HttpOnly || !cookie.Secure {
		t.Error("cleared ceremony cookie should keep HttpOnly and Secure")
	}
}

func TestSetWebAuthnSessionRequiresInit(t *testing.T) {
	saved := webAuthnCodec
	t.Cleanup(func() { webAuthnCodec = saved })
	webAuthnCodec = nil
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	if err := SetWebAuthnSession(&webauthn.SessionData{}, rec, req); err == nil {
		t.Error("expected error when webauthn is not initialised")
	}
}

package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/securecookie"
)

const webAuthnSessionCookie = "webauthn_session"

var webAuthn *webauthn.WebAuthn
var webAuthnCodec *securecookie.SecureCookie

func InitWebAuthn(rpOrigin, rpName, cookieSecret string) error {
	if rpName == "" {
		return fmt.Errorf("webauthn RP name is required")
	}

	origin, rpID, err := parseRelyingPartyOrigin(rpOrigin)
	if err != nil {
		return err
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpName,
		RPOrigins:     []string{origin},
	})
	if err != nil {
		return err
	}

	webAuthn = wa
	webAuthnCodec = newWebAuthnCodec(cookieSecret)
	return nil
}

func WebAuthn() *webauthn.WebAuthn {
	return webAuthn
}

func parseRelyingPartyOrigin(rpOrigin string) (origin string, rpID string, err error) {
	if rpOrigin == "" {
		return "", "", fmt.Errorf("webauthn RP origin is required")
	}
	u, err := url.Parse(rpOrigin)
	if err != nil {
		return "", "", fmt.Errorf("webauthn RP origin is not a valid URL: %w", err)
	}
	if u.Scheme == "" || u.Hostname() == "" {
		return "", "", fmt.Errorf("webauthn RP origin must include a scheme and host")
	}
	return u.Scheme + "://" + u.Host, u.Hostname(), nil
}

func newWebAuthnCodec(secret string) *securecookie.SecureCookie {
	hashKey := sha256.Sum256([]byte("webauthn-hash:" + secret))
	blockKey := sha256.Sum256([]byte("webauthn-block:" + secret))
	return securecookie.New(hashKey[:], blockKey[:])
}

func SetWebAuthnSession(session *webauthn.SessionData, w http.ResponseWriter, r *http.Request) error {
	if webAuthnCodec == nil {
		return fmt.Errorf("webauthn not initialised")
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	encoded, err := webAuthnCodec.Encode(webAuthnSessionCookie, raw)
	if err != nil {
		return err
	}

	maxAge := int(time.Until(session.Expires).Seconds())
	if maxAge < 1 {
		maxAge = 300
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnSessionCookie,
		Value:    encoded,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsHTTPS(r),
	})
	return nil
}

func ReadWebAuthnSession(r *http.Request) (*webauthn.SessionData, error) {
	if webAuthnCodec == nil {
		return nil, fmt.Errorf("webauthn not initialised")
	}
	cookie, err := r.Cookie(webAuthnSessionCookie)
	if err != nil {
		return nil, err
	}

	var raw []byte
	if err := webAuthnCodec.Decode(webAuthnSessionCookie, cookie.Value, &raw); err != nil {
		return nil, err
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func ClearWebAuthnSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Now().Add(-24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsHTTPS(r),
	})
}

package web

import (
	"net/http"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
)

func SetAuthCookie(userId string, w http.ResponseWriter, r *http.Request) error {
	secure := requestIsHTTPS(r)
	var cookie *http.Cookie
	if userId == "" {
		cookie = clearAuthCookie(secure)
	} else {
		var err error
		cookie, err = makeAuthCookie(userId, secure)
		if err != nil {
			return err
		}
	}

	http.SetCookie(w, cookie)
	return nil
}

func MakeAuthCookie(userId string) *http.Cookie {
	cookie, err := makeAuthCookie(userId, false)
	if err != nil {
		return nil
	}
	return cookie
}

func makeAuthCookie(userId string, secure bool) (*http.Cookie, error) {
	jwt, err := auth.MakeUserJwt(userId)
	if err != nil {
		return nil, err
	}
	ttl := auth.UserJwtTTL()
	expiration := time.Now().Add(ttl)
	return &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    jwt,
		Expires:  expiration,
		MaxAge:   int(ttl.Seconds()),
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Secure:   secure,
	}, nil
}

func clearAuthCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Secure:   secure,
	}
}

func requestIsHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

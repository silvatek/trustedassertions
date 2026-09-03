package web

import (
	"net/http"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
)

func SetAuthCookie(userId string, w http.ResponseWriter, r *http.Request) {
	secure := requestIsHTTPS(r)
	var cookie *http.Cookie
	if userId == "" {
		cookie = clearAuthCookie(secure)
	} else {
		cookie = makeAuthCookie(userId, secure)
	}

	http.SetCookie(w, cookie)
}

func MakeAuthCookie(userId string) *http.Cookie {
	return makeAuthCookie(userId, false)
}

func makeAuthCookie(userId string, secure bool) *http.Cookie {
	jwt, _ := auth.MakeUserJwt(userId, userJwtKey)
	expiration := time.Now().Add(2 * time.Hour)
	return &http.Cookie{
		Name:     "auth",
		Path:     "/",
		Value:    jwt,
		Expires:  expiration,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Secure:   secure,
	}
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

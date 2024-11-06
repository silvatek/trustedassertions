package web

import (
	"net/http"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
)

func SetAuthCookie(userId string, w http.ResponseWriter) {
	var cookie *http.Cookie
	if userId == "" {
		expiration := time.Now().Add(-24 * time.Hour)
		cookie = &http.Cookie{Name: "auth", Path: "/", Value: "", Expires: expiration, SameSite: http.SameSiteStrictMode}
	} else {
		cookie = MakeAuthCookie(userId)
	}

	http.SetCookie(w, cookie)
}

func MakeAuthCookie(userId string) *http.Cookie {
	jwt, _ := auth.MakeUserJwt(userId, userJwtKey)
	expiration := time.Now().Add(2 * time.Hour)
	cookie := http.Cookie{Name: "auth", Path: "/", Value: jwt, Expires: expiration, SameSite: http.SameSiteStrictMode}
	return &cookie
}

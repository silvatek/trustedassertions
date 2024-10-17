package web

import (
	"crypto/rand"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var userJwtKey []byte

func addAuthHandlers(r *mux.Router) {
	r.HandleFunc("/web/login", LoginWebHandler)
	r.HandleFunc("/web/logout", LogoutWebHandler)

	// Make a key for signing JWTs
	userJwtKey := make([]byte, 10)
	rand.Reader.Read(userJwtKey)
}

func LoginWebHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := ""
		if errorCode == "2001" {
			data = "Unable to verify identity"
		}

		RenderWebPage("loginform", data, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()
		userId := r.Form.Get("user_id")

		user, err := datastore.ActiveDataStore.FetchUser(userId)
		if err != nil {
			log.Errorf("User not found in login attempt: `%s`", userId)
			http.Redirect(w, r, "/web/login?err=2001", http.StatusSeeOther)
			return
		}
		if !user.CheckHash(r.Form.Get("password")) {
			log.Errorf("Incorrect password entered for: `%s`", userId)
			http.Redirect(w, r, "/web/login?err=2001", http.StatusSeeOther)
			return
		}

		cookie, _ := MakeAuthCookie(user)
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func MakeAuthCookie(user auth.User) (http.Cookie, error) {
	jwt, err := MakeUserJwt(user)
	expiration := time.Now().Add(1 * time.Hour)
	cookie := http.Cookie{Name: "auth", Path: "/", Value: jwt, Expires: expiration, SameSite: http.SameSiteStrictMode}
	return cookie, err
}

func MakeUserJwt(user auth.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": "trustedassertions",
			"sub": user.Id,
		})
	signed, err := token.SignedString(userJwtKey)
	if err != nil {
		return "", err
	}

	return signed, nil
}

func LogoutWebHandler(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{Name: "auth", Path: "/", Value: "", MaxAge: -1, SameSite: http.SameSiteStrictMode}
	http.SetCookie(w, &cookie)

	RenderWebPage("loggedout", "", w, r)
}

package web

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var userJwtKey []byte

func addAuthHandlers(r *mux.Router) {
	r.HandleFunc("/web/login", LoginWebHandler)
	r.HandleFunc("/web/logout", LogoutWebHandler)

	userJwtKey = auth.MakeJwtKey()
}

func LoginWebHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := ""
		if errorCode == strconv.Itoa(ErrorAuthFail) {
			data = "Unable to verify identity"
		}

		RenderWebPage("loginform", data, nil, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()
		userId := r.Form.Get("user_id")

		user, err := datastore.ActiveDataStore.FetchUser(userId)
		if err != nil {
			log.Errorf("User not found in login attempt: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail), http.StatusSeeOther)
			return
		}
		if !user.CheckHash(r.Form.Get("password")) {
			log.Errorf("Incorrect password entered for: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail), http.StatusSeeOther)
			return
		}

		cookie, _ := MakeAuthCookie(user)
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func MakeAuthCookie(user auth.User) (http.Cookie, error) {
	jwt, err := auth.MakeUserJwt(user, userJwtKey)
	expiration := time.Now().Add(1 * time.Hour)
	cookie := http.Cookie{Name: "auth", Path: "/", Value: jwt, Expires: expiration, SameSite: http.SameSiteStrictMode}
	return cookie, err
}

func LogoutWebHandler(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{Name: "auth", Path: "/", Value: "", MaxAge: -1, SameSite: http.SameSiteStrictMode}
	http.SetCookie(w, &cookie)

	RenderWebPage("loggedout", "", nil, w, r)
}

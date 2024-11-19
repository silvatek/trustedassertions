package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var userJwtKey []byte

func addAuthHandlers(r *mux.Router) {
	r.HandleFunc("/web/login", LoginWebHandler)
	r.HandleFunc("/web/logout", LogoutWebHandler)
	r.HandleFunc("/web/register", RegisterWebHandler)

	userJwtKey = auth.MakeJwtKey()
}

func LoginWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := ""
		if errorCode == strconv.Itoa(ErrorAuthFail) {
			data = "Unable to verify identity"
		}

		RenderWebPage(ctx, "loginform", data, nil, w, r)
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

		SetAuthCookie(userId, w)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func LogoutWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	cookie := http.Cookie{Name: "auth", Path: "/", Value: "", MaxAge: -1, SameSite: http.SameSiteStrictMode}
	http.SetCookie(w, &cookie)

	log.Debug("Cleared auth cookie")

	RenderWebPage(ctx, "loggedout", "", nil, w, r)
}

func RegisterWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := ""
		if errorCode == strconv.Itoa(ErrorRegCode) {
			data = "Registration code not valid"
		}

		RenderWebPage(ctx, "registrationform", data, nil, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()

		http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", ErrorRegCode), http.StatusSeeOther)
	}
}

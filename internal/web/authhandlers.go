package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

const AuthError = 3000

var ErrorNoAuth = AppError{ErrorCode: AuthError + 1, UserMessage: "Not logged in"}
var ErrorUserNotFound = AppError{ErrorCode: AuthError + 2, UserMessage: "User not found"}
var ErrorAuthFail = AppError{ErrorCode: AuthError + 5, UserMessage: "Not logged in"}
var ErrorRegCode = AppError{ErrorCode: AuthError + 101, UserMessage: "Registration code not valid"}

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
		if errorCode == strconv.Itoa(ErrorAuthFail.ErrorCode) {
			data = "Unable to verify identity"
		}

		RenderWebPage(ctx, "loginform", data, nil, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()
		userId := r.Form.Get("user_id")

		user, err := datastore.ActiveDataStore.FetchUser(ctx, userId)
		if err != nil {
			log.Errorf("User not found in login attempt: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail.ErrorCode), http.StatusSeeOther)
			return
		}
		if !user.CheckHash(r.Form.Get("password")) {
			log.Errorf("Incorrect password entered for: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail.ErrorCode), http.StatusSeeOther)
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

	log.DebugfX(ctx, "Cleared auth cookie")

	RenderWebPage(ctx, "loggedout", "", nil, w, r)
}

func RegisterWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := ""
		if errorCode == strconv.Itoa(ErrorRegCode.ErrorCode) {
			data = "Registration code not valid"
		}

		RenderWebPage(ctx, "registrationform", data, nil, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()

		code := r.Form.Get("reg_code")
		if code == "" {
			http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", ErrorRegCode.ErrorCode), http.StatusSeeOther)
			return
		}

		reg, err := datastore.ActiveDataStore.FetchRegistration(ctx, code)
		if err != nil {
			log.ErrorfX(ctx, "Could not load registration code %s, %v", code, err)
			http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", ErrorRegCode.ErrorCode), http.StatusSeeOther)
			return
		}
		if reg.Status != "Pending" {
			log.InfofX(ctx, "Attempt to reuse registration code %s (%s)", code, reg.Status)
			http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", ErrorRegCode.ErrorCode), http.StatusSeeOther)
			return
		}

		log.DebugfX(ctx, "Registering with valid registration code %s", code)

		user := auth.User{}

		user.Id = r.Form.Get("user_id")

		// Check for bad username
		// Check for duplicate username

		password := r.Form.Get("password1")
		//password2 := r.Form.Get("password2")
		// Check for password mismatch
		// Check for weak password

		user.HashPassword(password)

		// TODO: do all updates in a single transaction

		datastore.ActiveDataStore.StoreUser(ctx, user)

		reg.Code = code
		reg.UserName = user.Id
		reg.Status = "Complete"
		err = datastore.ActiveDataStore.StoreRegistration(ctx, reg)
		if err != nil {
			log.ErrorfX(ctx, "Error updating registration status: %v", err)
			http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", ErrorRegCode.ErrorCode), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

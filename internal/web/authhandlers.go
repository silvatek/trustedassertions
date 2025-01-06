package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nbutton23/zxcvbn-go"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/references"
)

const AuthError = 3000

var ErrorNoAuth = AppError{ErrorCode: AuthError + 1, UserMessage: "Not logged in"}
var ErrorUserNotFound = AppError{ErrorCode: AuthError + 2, UserMessage: "User not found"}
var ErrorAuthFail = AppError{ErrorCode: AuthError + 5, UserMessage: "Not logged in"}

const RegistrationError = 3100

var ErrorRegCode = AppError{ErrorCode: RegistrationError + 1, UserMessage: "Registration code not valid", HttpCode: 400}
var ErrorPasswordMismatch = AppError{ErrorCode: RegistrationError + 2, UserMessage: "Passwords do not match", HttpCode: 400}
var ErrorBadUsername = AppError{ErrorCode: RegistrationError + 3, UserMessage: "Username is not valid", HttpCode: 400}
var ErrorUserExists = AppError{ErrorCode: RegistrationError + 4, UserMessage: "Username already in use", HttpCode: 400}
var ErrorWeakPassword = AppError{ErrorCode: RegistrationError + 5, UserMessage: "Password is not strong enough", HttpCode: 400}
var ErrorRegistering = AppError{ErrorCode: RegistrationError + 6, UserMessage: "Unexpected error during registration"}

var RegistrationErrors = []AppError{ErrorRegCode, ErrorPasswordMismatch, ErrorBadUsername, ErrorUserExists, ErrorWeakPassword, ErrorRegistering}

var userJwtKey []byte

func addAuthHandlers(r *mux.Router) {
	r.HandleFunc("/web/login", LoginWebHandler)
	r.HandleFunc("/web/logout", LogoutWebHandler)
	r.HandleFunc("/web/register", RegisterWebHandler)
	r.HandleFunc("/web/profile", ProfileWebHandler)

	userJwtKey = auth.MakeJwtKey()
}

func nameOnly(username string) string {
	n := strings.Index(username, "@")
	if n == -1 {
		return username
	} else {
		return username[0:n]
	}
}

// Returns the name of the currently authenticated user, or an empty string.
func authUsername(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return ""
	}
	if cookie.Value == "" {
		return ""
	}
	userName, err := auth.ParseUserJwt(cookie.Value, userJwtKey)
	if err != nil {
		log.Errorf("Error parsing user JWT: %v", err)
		return ""
	}
	return userName
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

		http.Redirect(w, r, "/web", http.StatusSeeOther)
	}
}

func LogoutWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	cookie := http.Cookie{Name: "auth", Path: "/", Value: "", MaxAge: -1, SameSite: http.SameSiteStrictMode}
	http.SetCookie(w, &cookie)

	log.DebugfX(ctx, "Cleared auth cookie")

	RenderWebPage(ctx, "loggedout", "", nil, w, r)
}

type RegistrationStore interface {
	FetchUser(ctx context.Context, id string) (auth.User, error)
	StoreUser(ctx context.Context, user auth.User)
	FetchRegistration(ctx context.Context, code string) (auth.Registration, error)
	StoreRegistration(ctx context.Context, reg auth.Registration) error
}

type RegistrationForm struct {
	regCode   string
	userId    string
	password1 string
	password2 string
}

func RegisterWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	if r.Method == "GET" {
		errorCode := r.URL.Query().Get("err")

		data := "Error during registration"
		for _, error := range RegistrationErrors {
			if errorCode == strconv.Itoa(error.ErrorCode) {
				data = error.UserMessage
			}
		}

		RenderWebPage(ctx, "registrationform", data, nil, w, r)
	} else if r.Method == "POST" {
		r.ParseForm()

		registration := RegistrationForm{
			regCode:   r.Form.Get("reg_code"),
			userId:    r.Form.Get("user_id"),
			password1: r.Form.Get("password1"),
			password2: r.Form.Get("password2"),
		}

		err := registerUser(ctx, registration, datastore.ActiveDataStore)

		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("/web/register?err=%d", err.ErrorCode), http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/web/login", http.StatusSeeOther)
		}
	}
}

func registerUser(ctx context.Context, registration RegistrationForm, store RegistrationStore) *AppError {
	if registration.regCode == "" {
		return &ErrorRegCode
	}

	reg, err := store.FetchRegistration(ctx, registration.regCode)
	if err != nil {
		log.DebugfX(ctx, "Could not load registration code %s, %v", registration.regCode, err)
		return &ErrorRegCode
	}
	if reg.Status != "Pending" {
		log.DebugfX(ctx, "Attempt to reuse registration code %s (%s)", registration.regCode, reg.Status)
		return &ErrorRegCode
	}

	log.DebugfX(ctx, "Registering with valid registration code %s", registration.regCode)

	user := auth.User{}
	user.Id = registration.userId

	if len(user.Id) < 3 {
		return &ErrorBadUsername
	}

	if strings.ContainsAny(user.Id, "/:?") {
		return &ErrorBadUsername
	}

	_, err = store.FetchUser(ctx, user.Id)
	if err == nil {
		return &ErrorUserExists
	}

	if registration.password1 != registration.password2 {
		return &ErrorPasswordMismatch
	}

	strength := zxcvbn.PasswordStrength(registration.password1, []string{user.Id})
	if strength.Score < 3 {
		return &ErrorWeakPassword
	}

	user.HashPassword(registration.password1)

	reg.Code = registration.regCode
	reg.UserName = user.Id
	reg.Status = "Complete"
	err = store.StoreRegistration(ctx, reg)
	if err != nil {
		log.ErrorfX(ctx, "Error updating registration status: %v", err)
		return &ErrorRegistering
	}

	store.StoreUser(ctx, user)

	return nil
}

func ProfileWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return
	}

	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound, w, r)
		return
	}

	signers := make([]entities.Entity, len(user.KeyRefs))
	for n, keyRef := range user.KeyRefs {
		keyUri := references.UriFromString(keyRef.KeyId)
		entity, _ := datastore.ActiveDataStore.FetchEntity(ctx, keyUri)
		signers[n] = entity
	}

	data := struct {
		UserName string
		User     auth.User
		Entities []entities.Entity
	}{
		UserName: username,
		User:     user,
		Entities: signers,
	}

	RenderWebPage(ctx, "viewprofile", data, nil, w, r)
}

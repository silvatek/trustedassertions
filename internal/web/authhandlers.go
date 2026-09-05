package web

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
var ErrorForbidden = AppError{ErrorCode: AuthError + 3, UserMessage: "Not authorized", HttpCode: 403}
var ErrorAuthFail = AppError{ErrorCode: AuthError + 5, UserMessage: "Not logged in"}
var ErrorCreateInvite = AppError{ErrorCode: AuthError + 6, UserMessage: "Error creating invitation"}
var ErrorCreateSession = AppError{ErrorCode: AuthError + 7, UserMessage: "Unable to create session"}

const RegistrationError = 3100

var ErrorRegCode = AppError{ErrorCode: RegistrationError + 1, UserMessage: "Registration code not valid", HttpCode: 400}
var ErrorPasswordMismatch = AppError{ErrorCode: RegistrationError + 2, UserMessage: "Passwords do not match", HttpCode: 400}
var ErrorBadUsername = AppError{ErrorCode: RegistrationError + 3, UserMessage: "Username is not valid", HttpCode: 400}
var ErrorUserExists = AppError{ErrorCode: RegistrationError + 4, UserMessage: "Username already in use", HttpCode: 400}
var ErrorWeakPassword = AppError{ErrorCode: RegistrationError + 5, UserMessage: "Password is not strong enough", HttpCode: 400}
var ErrorRegistering = AppError{ErrorCode: RegistrationError + 6, UserMessage: "Unexpected error during registration"}

var RegistrationErrors = []AppError{ErrorRegCode, ErrorPasswordMismatch, ErrorBadUsername, ErrorUserExists, ErrorWeakPassword, ErrorRegistering}

// invalidRegCodeDelay slows responses for unknown or already-used codes so guessing is less practical.
var invalidRegCodeDelay = 300 * time.Millisecond

func addAuthHandlers(r *mux.Router) {
	r.HandleFunc("/web/login", LoginWebHandler)
	r.HandleFunc("/web/logout", LogoutWebHandler)
	r.HandleFunc("/web/register", RegisterWebHandler)
	r.HandleFunc("/web/profile", ProfileWebHandler)
	r.HandleFunc("/web/admin", AdminWebHandler)
	r.HandleFunc("/web/admin/invites", AdminInvitesWebHandler)
	r.HandleFunc("/web/admin/users", AdminUsersWebHandler)
	addPasskeyHandlers(r)
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
	userName, err := auth.ParseUserJwt(cookie.Value)
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
		if user.IsLocked() {
			log.Errorf("Locked user login attempt: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail.ErrorCode), http.StatusSeeOther)
			return
		}
		if !user.CheckHash(r.Form.Get("password")) {
			log.Errorf("Incorrect password entered for: `%s`", userId)
			http.Redirect(w, r, fmt.Sprintf("/web/login?err=%d", ErrorAuthFail.ErrorCode), http.StatusSeeOther)
			return
		}

		if err := SetAuthCookie(userId, w, r); err != nil {
			HandleError(ctx, ErrorCreateSession.instance(err.Error()), w, r)
			return
		}

		http.Redirect(w, r, HomePath, http.StatusSeeOther)
	}
}

func LogoutWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	if err := SetAuthCookie("", w, r); err != nil {
		log.ErrorfX(ctx, "Error clearing auth cookie: %v", err)
	}

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

		data := ""
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

func rejectInvalidRegCode() *AppError {
	time.Sleep(invalidRegCodeDelay)
	return &ErrorRegCode
}

func registerUser(ctx context.Context, registration RegistrationForm, store RegistrationStore) *AppError {
	code := auth.NormalizeInviteCode(registration.regCode)
	if code == "" {
		return rejectInvalidRegCode()
	}

	reg, err := store.FetchRegistration(ctx, code)
	if err != nil {
		log.DebugfX(ctx, "Could not load registration code %s, %v", code, err)
		return rejectInvalidRegCode()
	}
	if reg.Status != "Pending" {
		log.DebugfX(ctx, "Attempt to reuse registration code %s (%s)", code, reg.Status)
		return rejectInvalidRegCode()
	}
	if reg.IsExpired(time.Now()) {
		log.DebugfX(ctx, "Attempt to use expired registration code %s", code)
		return rejectInvalidRegCode()
	}

	log.DebugfX(ctx, "Registering with valid registration code %s", code)

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

	for _, role := range reg.Roles {
		user.AddRole(role)
	}

	reg.Code = code
	reg.UserName = user.Id
	reg.Status = "Complete"
	reg.CompletedAt = time.Now().UTC()
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

	passkeys := make([]passkeyView, len(user.Passkeys))
	for i, pk := range user.Passkeys {
		name := pk.Name
		if name == "" {
			name = "Passkey"
		}
		passkeys[i] = passkeyView{
			ID:         base64.RawURLEncoding.EncodeToString(pk.ID),
			Name:       name,
			CreatedAt:  formatPasskeyTime(pk.CreatedAt),
			LastUsedAt: formatPasskeyTime(pk.LastUsedAt),
		}
	}

	data := struct {
		UserName string
		User     auth.User
		Entities []entities.Entity
		Passkeys []passkeyView
	}{
		UserName: username,
		User:     user,
		Entities: signers,
		Passkeys: passkeys,
	}

	RenderWebPage(ctx, "viewprofile", data, nil, w, r)
}

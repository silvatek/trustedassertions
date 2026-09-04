package web

import (
	"encoding/base64"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	wa "github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

const passkeyLoginFailMessage = "Unable to verify identity"

func addPasskeyHandlers(r *mux.Router) {
	r.HandleFunc("/web/passkey/register/begin", PasskeyRegisterBeginHandler)
	r.HandleFunc("/web/passkey/register/finish", PasskeyRegisterFinishHandler)
	r.HandleFunc("/web/passkey/login/begin", PasskeyLoginBeginHandler)
	r.HandleFunc("/web/passkey/login/finish", PasskeyLoginFinishHandler)
	r.HandleFunc("/web/passkey/revoke", PasskeyRevokeHandler)
}

func PasskeyRegisterBeginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := loggedInUser(w, r)
	if !ok {
		return
	}
	if webAuthn == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "passkeys are not available")
		return
	}
	if len(user.Passkeys) >= auth.MaxPasskeys {
		writeJSONError(w, http.StatusConflict, "user already has the maximum number of passkeys")
		return
	}

	creation, session, err := webAuthn.BeginRegistration(
		&user,
		wa.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		}),
		wa.WithExclusions(excludeCredentials(user)),
	)
	if err != nil {
		log.Errorf("BeginRegistration: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not start passkey registration")
		return
	}

	if err := SetWebAuthnSession(session, w, r); err != nil {
		log.Errorf("SetWebAuthnSession: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not start passkey registration")
		return
	}

	writeJSON(w, http.StatusOK, creation)
}

func PasskeyRegisterFinishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := loggedInUser(w, r)
	if !ok {
		return
	}
	if webAuthn == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "passkeys are not available")
		return
	}

	session, err := ReadWebAuthnSession(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "passkey registration is not in progress")
		return
	}
	defer ClearWebAuthnSession(w, r)

	if string(session.UserID) != user.Id {
		writeJSONError(w, http.StatusBadRequest, "passkey registration is not in progress")
		return
	}

	cred, err := webAuthn.FinishRegistration(&user, *session, r)
	if err != nil {
		log.Errorf("FinishRegistration: %v", err)
		writeJSONError(w, http.StatusBadRequest, "could not register passkey")
		return
	}

	pk := auth.PasskeyFromCredential(*cred)
	if err := datastore.AddPasskey(appcontext.NewWebContext(r), user.Id, pk); err != nil {
		log.Errorf("AddPasskey: %v", err)
		writeJSONError(w, http.StatusBadRequest, "could not register passkey")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func PasskeyLoginBeginHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getPasskeyLoginUser(w, r)
	if !ok {
		return
	}

	assertion, session, err := webAuthn.BeginLogin(
		&user,
		wa.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		log.Errorf("BeginLogin: %v", err)
		writePasskeyLoginFailure(w)
		return
	}

	writePasskeyLoginBeginResponse(w, r, assertion, session)
}

func PasskeyLoginFinishHandler(w http.ResponseWriter, r *http.Request) {
	user, session, ok := getPasskeyLoginSession(w, r)
	if !ok {
		return
	}
	defer ClearWebAuthnSession(w, r)

	cred, err := webAuthn.FinishLogin(&user, *session, r)
	if err != nil {
		log.Errorf("FinishLogin: %v", err)
		writePasskeyLoginFailure(w)
		return
	}

	writePasskeyLoginFinishResponse(w, r, user, cred)
}

func PasskeyRevokeHandler(w http.ResponseWriter, r *http.Request) {
	user, credentialID, ok := getPasskeyRevokeRequest(w, r)
	if !ok {
		return
	}

	if err := datastore.RemovePasskey(appcontext.NewWebContext(r), user.Id, credentialID); err != nil {
		log.Errorf("RemovePasskey: %v", err)
		writePasskeyRevokeFailure(w, err)
		return
	}

	writePasskeyRevokeResponse(w)
}

func getPasskeyLoginUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if !requirePasskeyPOST(w, r) {
		return auth.User{}, false
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writePasskeyLoginFailure(w)
		return auth.User{}, false
	}
	return fetchPasskeyLoginUser(w, r, body.UserID)
}

func writePasskeyLoginBeginResponse(w http.ResponseWriter, r *http.Request, assertion *protocol.CredentialAssertion, session *wa.SessionData) {
	if err := SetWebAuthnSession(session, w, r); err != nil {
		log.Errorf("SetWebAuthnSession: %v", err)
		writePasskeyLoginFailure(w)
		return
	}
	writeJSON(w, http.StatusOK, assertion)
}

func getPasskeyLoginSession(w http.ResponseWriter, r *http.Request) (auth.User, *wa.SessionData, bool) {
	if !requirePasskeyPOST(w, r) {
		return auth.User{}, nil, false
	}

	session, err := ReadWebAuthnSession(r)
	if err != nil {
		writePasskeyLoginFailure(w)
		return auth.User{}, nil, false
	}

	user, ok := fetchPasskeyLoginUser(w, r, string(session.UserID))
	if !ok {
		return auth.User{}, nil, false
	}
	if string(session.UserID) != user.Id {
		writePasskeyLoginFailure(w)
		return auth.User{}, nil, false
	}
	return user, session, true
}

func writePasskeyLoginFinishResponse(w http.ResponseWriter, r *http.Request, user auth.User, cred *wa.Credential) {
	if err := datastore.RecordPasskeyUse(appcontext.NewWebContext(r), user.Id, auth.PasskeyFromCredential(*cred), time.Now().UTC()); err != nil {
		log.Errorf("RecordPasskeyUse: %v", err)
		writePasskeyLoginFailure(w)
		return
	}
	if cred.Authenticator.CloneWarning {
		log.Errorf("passkey clone warning for user %s", user.Id)
		writePasskeyLoginFailure(w)
		return
	}

	SetAuthCookie(user.Id, w, r)
	writeJSON(w, http.StatusOK, map[string]string{"redirect": HomePath})
}

func getPasskeyRevokeRequest(w http.ResponseWriter, r *http.Request) (auth.User, []byte, bool) {
	if !requirePOST(w, r) {
		return auth.User{}, nil, false
	}

	user, ok := loggedInUser(w, r)
	if !ok {
		return auth.User{}, nil, false
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid passkey id")
		return auth.User{}, nil, false
	}

	credentialID, err := base64.RawURLEncoding.DecodeString(body.ID)
	if err != nil || len(credentialID) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid passkey id")
		return auth.User{}, nil, false
	}
	return user, credentialID, true
}

func writePasskeyRevokeFailure(w http.ResponseWriter, err error) {
	if stderrors.Is(err, auth.ErrPasskeyNotFound) {
		writeJSONError(w, http.StatusNotFound, "passkey not found")
		return
	}
	writeJSONError(w, http.StatusInternalServerError, "could not revoke passkey")
}

func writePasskeyRevokeResponse(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func requirePOST(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

func requirePasskeyPOST(w http.ResponseWriter, r *http.Request) bool {
	if !requirePOST(w, r) {
		return false
	}
	if webAuthn == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "passkeys are not available")
		return false
	}
	return true
}

func fetchPasskeyLoginUser(w http.ResponseWriter, r *http.Request, userID string) (auth.User, bool) {
	if userID == "" {
		writePasskeyLoginFailure(w)
		return auth.User{}, false
	}
	user, err := datastore.ActiveDataStore.FetchUser(appcontext.NewWebContext(r), userID)
	if err != nil || len(user.Passkeys) == 0 {
		writePasskeyLoginFailure(w)
		return auth.User{}, false
	}
	return user, true
}

func writePasskeyLoginFailure(w http.ResponseWriter) {
	writeJSONError(w, http.StatusUnauthorized, passkeyLoginFailMessage)
}

func loggedInUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	username := authUsername(r)
	if username == "" {
		writeJSONError(w, http.StatusUnauthorized, "not logged in")
		return auth.User{}, false
	}
	user, err := datastore.ActiveDataStore.FetchUser(appcontext.NewWebContext(r), username)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "not logged in")
		return auth.User{}, false
	}
	return user, true
}

func excludeCredentials(user auth.User) []protocol.CredentialDescriptor {
	list := make([]protocol.CredentialDescriptor, len(user.Passkeys))
	for i, pk := range user.Passkeys {
		list[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: pk.ID,
		}
	}
	return list
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type passkeyView struct {
	Name       string
	CreatedAt  string
	LastUsedAt string
}

func formatPasskeyTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.UTC().Format("2006-01-02")
}

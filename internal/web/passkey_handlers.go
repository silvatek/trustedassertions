package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	wa "github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func addPasskeyHandlers(r *mux.Router) {
	r.HandleFunc("/web/passkey/register/begin", PasskeyRegisterBeginHandler)
	r.HandleFunc("/web/passkey/register/finish", PasskeyRegisterFinishHandler)
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

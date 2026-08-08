package auth

import (
	"bytes"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	log "silvatek.uk/trustedassertions/internal/logging"
)

type PasskeyUser struct {
	Name string
}

func (u *PasskeyUser) WebAuthnCredentials() []webauthn.Credential {
	return nil
}

func (u *PasskeyUser) WebAuthnID() []byte {
	return []byte(u.Name)
}

func (u *PasskeyUser) WebAuthnName() string {
	return u.Name
}

func (u *PasskeyUser) WebAuthnDisplayName() string {
	return u.Name
}

type PasskeyData struct {
	Username string
	Status   string
	Options  *protocol.CredentialCreation
}

var webAuthn *webauthn.WebAuthn = nil

func InitWebAuthn() {
	if webAuthn == nil {
		wconfig := &webauthn.Config{
			RPDisplayName: "Trusted Assertion",                // Display Name for your site
			RPID:          "localhost",                        // Generally the FQDN for your site
			RPOrigins:     []string{"http://localhost:8080/"}, // The origin URLs allowed for WebAuthn
		}

		var err error
		if webAuthn, err = webauthn.New(wconfig); err != nil {
			log.Errorf("Error initialising webauthn: %s", err)
		}
	}
}

func BeginPasskeyRegistration() PasskeyData {
	log.Info("Begin passkey registration")

	InitWebAuthn()

	user := PasskeyUser{Name: "Sam"}

	options, _, err := webAuthn.BeginRegistration(&user)
	if err != nil {
		log.Errorf("Error beginning webauthn registration: %s", err)
		return PasskeyData{Status: "Error"}
	}

	return PasskeyData{
		Status:   "OK",
		Username: user.Name,
		Options:  options,
	}
}

func FinishPasskeyRegistration(r *http.Request) {
	log.Info("Finish passkey registration")

	InitWebAuthn()

	buf := new(bytes.Buffer)

	buf.ReadFrom(r.Body)
	log.Info(buf.String())

	// session := webauthn.SessionData{}

	// webAuthn.FinishRegistration(nil, session, r)
}

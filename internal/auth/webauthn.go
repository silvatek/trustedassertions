package auth

import "github.com/go-webauthn/webauthn/webauthn"

var _ webauthn.User = (*User)(nil)

func (u *User) WebAuthnID() []byte {
	return []byte(u.Id)
}

func (u *User) WebAuthnName() string {
	return u.Id
}

func (u *User) WebAuthnDisplayName() string {
	return u.Id
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(u.Passkeys))
	for i, pk := range u.Passkeys {
		creds[i] = pk.Credential()
	}
	return creds
}

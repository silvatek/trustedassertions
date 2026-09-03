package auth

import (
	"bytes"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const MaxPasskeys = 5

var (
	ErrEmptyPasskeyID  = errors.New("passkey credential id is required")
	ErrTooManyPasskeys = errors.New("user already has the maximum number of passkeys")
	ErrPasskeyExists   = errors.New("passkey already registered")
	ErrPasskeyNotFound = errors.New("passkey not found")
)

type Passkey struct {
	ID              []byte    `json:"id"`
	PublicKey       []byte    `json:"publicKey"`
	AttestationType string    `json:"attestationType"`
	Transports      []string  `json:"transports"`
	UserPresent     bool      `json:"userPresent"`
	UserVerified    bool      `json:"userVerified"`
	BackupEligible  bool      `json:"backupEligible"`
	BackupState     bool      `json:"backupState"`
	AAGUID          []byte    `json:"aaguid"`
	SignCount       uint32    `json:"signCount"`
	CloneWarning    bool      `json:"cloneWarning"`
	Attachment      string    `json:"attachment"`
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"createdAt"`
	LastUsedAt      time.Time `json:"lastUsedAt"`
}

func PasskeyFromCredential(cred webauthn.Credential, name string) Passkey {
	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}
	return Passkey{
		ID:              cred.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transports:      transports,
		UserPresent:     cred.Flags.UserPresent,
		UserVerified:    cred.Flags.UserVerified,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		AAGUID:          cred.Authenticator.AAGUID,
		SignCount:       cred.Authenticator.SignCount,
		CloneWarning:    cred.Authenticator.CloneWarning,
		Attachment:      string(cred.Authenticator.Attachment),
		Name:            name,
	}
}

func (p Passkey) Credential() webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, len(p.Transports))
	for i, t := range p.Transports {
		transports[i] = protocol.AuthenticatorTransport(t)
	}
	return webauthn.Credential{
		ID:              p.ID,
		PublicKey:       p.PublicKey,
		AttestationType: p.AttestationType,
		Transport:       transports,
		Flags:           webauthn.NewCredentialFlags(p.authenticatorFlags()),
		Authenticator: webauthn.Authenticator{
			AAGUID:       p.AAGUID,
			SignCount:    p.SignCount,
			CloneWarning: p.CloneWarning,
			Attachment:   protocol.AuthenticatorAttachment(p.Attachment),
		},
	}
}

func (p Passkey) authenticatorFlags() protocol.AuthenticatorFlags {
	var flags protocol.AuthenticatorFlags
	if p.UserPresent {
		flags |= protocol.FlagUserPresent
	}
	if p.UserVerified {
		flags |= protocol.FlagUserVerified
	}
	if p.BackupEligible {
		flags |= protocol.FlagBackupEligible
	}
	if p.BackupState {
		flags |= protocol.FlagBackupState
	}
	return flags
}

func (u *User) AddPasskey(pk Passkey) error {
	if len(pk.ID) == 0 {
		return ErrEmptyPasskeyID
	}
	if len(u.Passkeys) >= MaxPasskeys {
		return ErrTooManyPasskeys
	}
	for _, existing := range u.Passkeys {
		if bytes.Equal(existing.ID, pk.ID) {
			return ErrPasskeyExists
		}
	}
	if pk.CreatedAt.IsZero() {
		pk.CreatedAt = time.Now().UTC()
	}
	u.Passkeys = append(u.Passkeys, pk)
	return nil
}

func (u *User) RemovePasskey(credentialID []byte) error {
	if len(credentialID) == 0 {
		return ErrEmptyPasskeyID
	}
	kept := make([]Passkey, 0, len(u.Passkeys))
	found := false
	for _, pk := range u.Passkeys {
		if bytes.Equal(pk.ID, credentialID) {
			found = true
			continue
		}
		kept = append(kept, pk)
	}
	if !found {
		return ErrPasskeyNotFound
	}
	u.Passkeys = kept
	return nil
}

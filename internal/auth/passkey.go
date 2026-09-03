package auth

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Well-known authenticator model names keyed by AAGUID.
// See https://github.com/passkeydeveloper/passkey-authenticator-aaguids
var aaguidNames = map[string]string{
	"fbfc3007-154e-4ecc-8c0b-6e020557d7bd": "iCloud Keychain",
	"dd4ec289-e01d-41c9-bb89-70fa845d4bf2": "iCloud Keychain",
	"ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4": "Google Password Manager",
	"adce0002-35bc-c60a-648b-0b25f1f05503": "Chrome",
	"b5397666-4898-4ae1-b2c1-3a7b4d51c2dd": "Chrome",
	"c5efc2ca-7dc0-4fa5-8cbf-0bb7e054c4c2": "Chrome",
	"08987058-cadc-4b81-b6e1-30de50dcbe96": "Windows Hello",
	"9ddd1817-af5a-4672-a2b9-3e3dd95000a9": "Android",
	"2fc0579f-8113-47ea-b116-bb5a8db9202a": "YubiKey 5",
	"fa2b99dc-9e39-4257-8f92-4a30d23c4118": "YubiKey 5 NFC",
	"cb69481e-8ff7-4039-93ec-0a2729a154a8": "YubiKey 5Ci",
	"ee882879-721c-4913-9775-3dfcce97072a": "YubiKey 5C",
}

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

func PasskeyFromCredential(cred webauthn.Credential) Passkey {
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
		Name:            derivedPasskeyName(cred),
		CreatedAt:       time.Now().UTC(),
	}
}

func derivedPasskeyName(cred webauthn.Credential) string {
	if name := nameForAAGUID(cred.Authenticator.AAGUID); name != "" {
		return name
	}
	if hasTransport(cred.Transport, protocol.USB) || hasTransport(cred.Transport, protocol.NFC) || hasTransport(cred.Transport, protocol.BLE) {
		return "Security key"
	}
	if hasTransport(cred.Transport, protocol.Hybrid) {
		return "Phone"
	}
	platform := cred.Authenticator.Attachment == protocol.Platform || hasTransport(cred.Transport, protocol.Internal)
	synced := cred.Flags.BackupEligible || cred.Flags.BackupState
	if platform && synced {
		return "Synced passkey"
	}
	if platform {
		return "This device"
	}
	if synced {
		return "Synced passkey"
	}
	return "Passkey"
}

func nameForAAGUID(id []byte) string {
	if len(id) != 16 || bytes.Equal(id, make([]byte, 16)) {
		return ""
	}
	key := fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
	return aaguidNames[key]
}

func hasTransport(transports []protocol.AuthenticatorTransport, want protocol.AuthenticatorTransport) bool {
	for _, t := range transports {
		if t == want {
			return true
		}
	}
	return false
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

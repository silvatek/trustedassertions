package auth

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func samplePasskey(id string) Passkey {
	return Passkey{
		ID:              []byte(id),
		PublicKey:       []byte("pubkey-" + id),
		AttestationType: "none",
		Transports:      []string{"internal", "hybrid"},
		UserPresent:     true,
		UserVerified:    true,
		BackupEligible:  true,
		BackupState:     false,
		AAGUID:          []byte{1, 2, 3, 4},
		SignCount:       7,
		Name:            "Laptop",
		CreatedAt:       time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
	}
}

func TestPasskeyRecordUse(t *testing.T) {
	pk := samplePasskey("cred-1")
	created := pk.CreatedAt
	usedAt := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	update := Passkey{
		ID:             pk.ID,
		SignCount:      42,
		CloneWarning:   true,
		UserPresent:    false,
		UserVerified:   true,
		BackupEligible: false,
		BackupState:    true,
		Name:           "should not apply",
		CreatedAt:      time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	pk.RecordUse(update, usedAt)

	if pk.SignCount != 42 {
		t.Errorf("sign count = %d", pk.SignCount)
	}
	if !pk.CloneWarning {
		t.Error("clone warning should be updated")
	}
	if pk.UserPresent {
		t.Error("user present should be updated")
	}
	if !pk.UserVerified {
		t.Error("user verified should be updated")
	}
	if pk.BackupEligible {
		t.Error("backup eligible should be updated")
	}
	if !pk.BackupState {
		t.Error("backup state should be updated")
	}
	if !pk.LastUsedAt.Equal(usedAt) {
		t.Errorf("last used = %v", pk.LastUsedAt)
	}
	if pk.Name != "Laptop" {
		t.Errorf("name = %q, should be unchanged", pk.Name)
	}
	if !pk.CreatedAt.Equal(created) {
		t.Errorf("created at = %v, should be unchanged", pk.CreatedAt)
	}
}

func TestUserRecordPasskeyUseUnknownID(t *testing.T) {
	user := User{Id: "alice"}
	if err := user.AddPasskey(samplePasskey("cred-1")); err != nil {
		t.Fatal(err)
	}
	if err := user.RecordPasskeyUse(samplePasskey("other"), time.Now()); !errors.Is(err, ErrPasskeyNotFound) {
		t.Errorf("unknown id error = %v", err)
	}
	if err := user.RecordPasskeyUse(Passkey{}, time.Now()); !errors.Is(err, ErrEmptyPasskeyID) {
		t.Errorf("empty id error = %v", err)
	}
}

func TestUserRecordPasskeyUse(t *testing.T) {
	user := User{Id: "alice"}
	pk := samplePasskey("cred-1")
	if err := user.AddPasskey(pk); err != nil {
		t.Fatal(err)
	}

	usedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	update := samplePasskey("cred-1")
	update.SignCount = 99
	update.CloneWarning = true
	update.Name = "ignored"
	update.CreatedAt = time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := user.RecordPasskeyUse(update, usedAt); err != nil {
		t.Fatalf("RecordPasskeyUse: %v", err)
	}
	got := user.Passkeys[0]
	if got.SignCount != 99 {
		t.Errorf("sign count = %d", got.SignCount)
	}
	if !got.CloneWarning {
		t.Error("clone warning should be updated")
	}
	if got.Name != "Laptop" {
		t.Errorf("name = %q, should be unchanged", got.Name)
	}
	if !got.CreatedAt.Equal(pk.CreatedAt) {
		t.Errorf("created at = %v, should be unchanged", got.CreatedAt)
	}
	if !got.LastUsedAt.Equal(usedAt) {
		t.Errorf("last used = %v", got.LastUsedAt)
	}
}

func TestAddRemovePasskey(t *testing.T) {
	user := User{Id: "alice"}
	pk := samplePasskey("cred-1")

	if err := user.AddPasskey(pk); err != nil {
		t.Fatalf("AddPasskey: %v", err)
	}
	if len(user.Passkeys) != 1 {
		t.Fatalf("passkey count = %d", len(user.Passkeys))
	}

	if err := user.AddPasskey(pk); !errors.Is(err, ErrPasskeyExists) {
		t.Errorf("duplicate AddPasskey error = %v", err)
	}

	if err := user.RemovePasskey([]byte("cred-1")); err != nil {
		t.Fatalf("RemovePasskey: %v", err)
	}
	if len(user.Passkeys) != 0 {
		t.Errorf("passkeys after remove = %d", len(user.Passkeys))
	}
	if err := user.RemovePasskey([]byte("cred-1")); !errors.Is(err, ErrPasskeyNotFound) {
		t.Errorf("missing RemovePasskey error = %v", err)
	}
}

func TestAddPasskeyEmptyID(t *testing.T) {
	user := User{Id: "alice"}
	if err := user.AddPasskey(Passkey{}); !errors.Is(err, ErrEmptyPasskeyID) {
		t.Errorf("error = %v", err)
	}
}

func TestAddPasskeyCap(t *testing.T) {
	user := User{Id: "alice"}
	for i := 0; i < MaxPasskeys; i++ {
		if err := user.AddPasskey(samplePasskey(string(rune('a' + i)))); err != nil {
			t.Fatalf("AddPasskey %d: %v", i, err)
		}
	}
	if err := user.AddPasskey(samplePasskey("overflow")); !errors.Is(err, ErrTooManyPasskeys) {
		t.Errorf("cap error = %v", err)
	}
}

func TestPasskeyCredentialRoundTrip(t *testing.T) {
	cred := webauthn.Credential{
		ID:              []byte("cred-1"),
		PublicKey:       []byte{9, 8, 7},
		AttestationType: "none",
		Transport:       []protocol.AuthenticatorTransport{"internal"},
		Flags:           webauthn.NewCredentialFlags(protocol.FlagUserPresent | protocol.FlagUserVerified),
		Authenticator: webauthn.Authenticator{
			AAGUID:     []byte{1, 2},
			SignCount:  11,
			Attachment: protocol.Platform,
		},
	}

	pk := PasskeyFromCredential(cred)
	got := pk.Credential()

	if !bytes.Equal(got.ID, cred.ID) || !bytes.Equal(got.PublicKey, cred.PublicKey) {
		t.Error("id or public key did not round-trip")
	}
	if got.AttestationType != cred.AttestationType {
		t.Errorf("attestation type = %q", got.AttestationType)
	}
	if got.Authenticator.SignCount != 11 {
		t.Errorf("sign count = %d", got.Authenticator.SignCount)
	}
	if !got.Flags.UserPresent || !got.Flags.UserVerified {
		t.Error("flags did not round-trip")
	}
	if pk.Name != "This device" {
		t.Errorf("name = %q", pk.Name)
	}
	if pk.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set when the passkey is created")
	}
}

func TestUserPasskeyJSONRoundTrip(t *testing.T) {
	user := User{Id: "alice", PassHash: "hash"}
	if err := user.AddPasskey(samplePasskey("cred-1")); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got User
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id != "alice" || got.PassHash != "hash" {
		t.Errorf("user fields = %+v", got)
	}
	if len(got.Passkeys) != 1 {
		t.Fatalf("passkeys = %d", len(got.Passkeys))
	}
	if !bytes.Equal(got.Passkeys[0].ID, []byte("cred-1")) {
		t.Errorf("credential id = %q", got.Passkeys[0].ID)
	}
	if !bytes.Equal(got.Passkeys[0].PublicKey, []byte("pubkey-cred-1")) {
		t.Errorf("public key = %q", got.Passkeys[0].PublicKey)
	}
	if got.Passkeys[0].SignCount != 7 {
		t.Errorf("sign count = %d", got.Passkeys[0].SignCount)
	}
}

func TestWebAuthnCredentialsFromPasskeys(t *testing.T) {
	user := &User{Id: "alice"}
	if err := user.AddPasskey(samplePasskey("cred-1")); err != nil {
		t.Fatal(err)
	}
	creds := user.WebAuthnCredentials()
	if len(creds) != 1 {
		t.Fatalf("credentials = %d", len(creds))
	}
	if !bytes.Equal(creds[0].ID, []byte("cred-1")) {
		t.Errorf("credential id = %q", creds[0].ID)
	}
}

func TestDerivedPasskeyName(t *testing.T) {
	icloud, err := hex.DecodeString("fbfc3007154e4ecc8c0b6e020557d7bd")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		cred webauthn.Credential
		want string
	}{
		{
			name: "known aaguid",
			cred: webauthn.Credential{Authenticator: webauthn.Authenticator{AAGUID: icloud}},
			want: "iCloud Keychain",
		},
		{
			name: "security key",
			cred: webauthn.Credential{Transport: []protocol.AuthenticatorTransport{protocol.USB}},
			want: "Security key",
		},
		{
			name: "phone",
			cred: webauthn.Credential{Transport: []protocol.AuthenticatorTransport{protocol.Hybrid}},
			want: "Phone",
		},
		{
			name: "this device",
			cred: webauthn.Credential{
				Transport:     []protocol.AuthenticatorTransport{protocol.Internal},
				Authenticator: webauthn.Authenticator{Attachment: protocol.Platform},
			},
			want: "This device",
		},
		{
			name: "synced platform",
			cred: webauthn.Credential{
				Transport:     []protocol.AuthenticatorTransport{protocol.Internal},
				Flags:         webauthn.NewCredentialFlags(protocol.FlagBackupEligible),
				Authenticator: webauthn.Authenticator{Attachment: protocol.Platform},
			},
			want: "Synced passkey",
		},
		{
			name: "unknown",
			cred: webauthn.Credential{},
			want: "Passkey",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := derivedPasskeyName(tc.cred)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

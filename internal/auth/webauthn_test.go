package auth

import (
	"bytes"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestUserImplementsWebAuthnUser(t *testing.T) {
	user := &User{Id: "alice"}

	var wu webauthn.User = user
	if !bytes.Equal(wu.WebAuthnID(), []byte("alice")) {
		t.Errorf("WebAuthnID = %q, want alice", wu.WebAuthnID())
	}
	if wu.WebAuthnName() != "alice" {
		t.Errorf("WebAuthnName = %q, want alice", wu.WebAuthnName())
	}
	if wu.WebAuthnDisplayName() != "alice" {
		t.Errorf("WebAuthnDisplayName = %q, want alice", wu.WebAuthnDisplayName())
	}
	if wu.WebAuthnCredentials() == nil {
		t.Error("WebAuthnCredentials should be an empty list until credentials are persisted")
	}
	if len(wu.WebAuthnCredentials()) != 0 {
		t.Errorf("WebAuthnCredentials length = %d, want 0", len(wu.WebAuthnCredentials()))
	}
}

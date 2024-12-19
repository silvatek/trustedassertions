package auth

import (
	"bytes"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/logging"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func TestPasswordHash(t *testing.T) {
	user := User{Id: "x"}

	if user.CheckHash("somerandompassword") {
		t.Error("Should not match empty password")
	}

	user.HashPassword("somerandompassword")

	if user.PassHash == "" {
		t.Error("Password hash not found")
	}

	if len(user.PassHash) < 80 {
		t.Errorf("Password hash too short: %s", user.PassHash)
	}

	if !user.CheckHash("somerandompassword") {
		t.Error("Should match correct password")
	}

	if user.CheckHash("someotherpassword") {
		t.Error("Should not match incorrect password")
	}
}

func TestPasswordHashError(t *testing.T) {
	og := logging.LogWriter
	var buf bytes.Buffer
	log.LogWriter = &buf

	saved := DefaultHashCost
	DefaultHashCost = 99999

	user := User{Id: "x"}
	user.HashPassword("testing")

	if user.PassHash != "" {
		t.Error("User should not have a hashed password after error")
	}

	if !strings.HasPrefix(buf.String(), "ERROR Error hashing password") {
		t.Error("Did not find expected log message")
	}

	log.LogWriter = og

	DefaultHashCost = saved
}

func TestPasswordCheckError(t *testing.T) {
	user := User{Id: "x", PassHash: "&&&"} // Password hash is not valid Base64 encoding
	if user.CheckHash("test") {
		t.Error("Bad password hash should not validate")
	}
}

func TestKeyRefs(t *testing.T) {
	user := User{Id: "x"}

	if user.KeyRefs != nil {
		t.Error("Expected nil keyrefs initially")
	}

	if user.HasKey("abc") {
		t.Error("Expected user to not have key initially")
	}

	user.AddKeyRef("abc", "b")

	if user.KeyRefs == nil {
		t.Error("Expected non-nil keyrefs after addition")
	}

	if !user.HasKey("abc") {
		t.Error("Expected user to have key after addition")
	}

	if user.HasKey("xyz") {
		t.Error("Expected user to not have different key after addition")
	}
}

func TestParseBadJwt(t *testing.T) {
	jwt, err := ParseUserJwt("broken", []byte("badkey"))

	if err == nil {
		t.Error("Parsing broken JWT did not return an error")
	}
	if jwt != "" {
		t.Error("Parsing broken JWT did not return an empty string")
	}
}

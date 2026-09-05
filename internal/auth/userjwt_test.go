package auth

import (
	"strings"
	"testing"
	"time"
)

const testJwtKey = "shared-user-jwt-key-32-bytes-ok!"

func restoreUserJwt(t *testing.T) {
	t.Helper()
	prevKey := append([]byte(nil), userJwtKey...)
	prevTTL := userJwtTTL
	t.Cleanup(func() {
		userJwtKey = prevKey
		userJwtTTL = prevTTL
	})
}

func TestJwtSameKeyAcceptedByOtherProcess(t *testing.T) {
	processA := []byte(testJwtKey)
	processB := []byte(testJwtKey)

	token, err := makeUserJwt("Tester", processA)
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}

	username, err := parseUserJwt(token, processB)
	if err != nil {
		t.Fatalf("Error parsing JWT with the same key: %v", err)
	}
	if username != "Tester" {
		t.Errorf("Unexpected username: %s", username)
	}
}

func TestJwtDifferentKeysRejected(t *testing.T) {
	token, err := makeUserJwt("Tester", []byte(testJwtKey))
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}

	username, err := parseUserJwt(token, []byte("different-user-jwt-key-32-bytes!!"))
	if err == nil {
		t.Error("Parsing JWT with a different key did not return an error")
	}
	if username != "" {
		t.Errorf("Expected empty username, got %q", username)
	}
}

func TestJwtExpiresAfterTTL(t *testing.T) {
	restoreUserJwt(t)
	if err := InitUserJwt(testJwtKey, time.Second); err != nil {
		t.Fatalf("InitUserJwt: %v", err)
	}

	token, err := MakeUserJwt("Tester")
	if err != nil {
		t.Fatalf("Error making JWT: %v", err)
	}

	time.Sleep(2 * time.Second)

	username, err := ParseUserJwt(token)
	if err == nil {
		t.Error("Expected expired JWT to fail parse")
	}
	if username != "" {
		t.Errorf("Expected empty username after expiry, got %q", username)
	}
}

func TestInitUserJwtFromEnvUsesDefaultLocally(t *testing.T) {
	restoreUserJwt(t)
	t.Setenv("USER_JWT_KEY", "")
	t.Setenv("USER_JWT_TTL", "")
	t.Setenv("GCLOUD_PROJECT", "")

	if err := InitUserJwtFromEnv(); err != nil {
		t.Fatalf("InitUserJwtFromEnv: %v", err)
	}
	if string(userJwtKey) != defaultUserJwtKey {
		t.Errorf("userJwtKey = %q, want default", userJwtKey)
	}

	token, err := MakeUserJwt("Tester")
	if err != nil {
		t.Fatalf("MakeUserJwt with local default: %v", err)
	}
	username, err := ParseUserJwt(token)
	if err != nil {
		t.Fatalf("ParseUserJwt: %v", err)
	}
	if username != "Tester" {
		t.Errorf("username = %q, want Tester", username)
	}
}

func TestInitUserJwtFromEnvAllowsMissingKeyOnGcloud(t *testing.T) {
	restoreUserJwt(t)
	t.Setenv("USER_JWT_KEY", "")
	t.Setenv("USER_JWT_TTL", "")
	t.Setenv("GCLOUD_PROJECT", "trustedassertions")

	if err := InitUserJwtFromEnv(); err != nil {
		t.Fatalf("InitUserJwtFromEnv: %v", err)
	}
	if len(userJwtKey) != 0 {
		t.Errorf("userJwtKey should be empty when USER_JWT_KEY is unset")
	}
	if UserJwtTTL() != DefaultUserJwtTTL {
		t.Errorf("TTL = %v, want %v", UserJwtTTL(), DefaultUserJwtTTL)
	}

	token, err := MakeUserJwt("Tester")
	if err == nil {
		t.Fatal("expected MakeUserJwt to fail when USER_JWT_KEY is unset")
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
	if !strings.Contains(err.Error(), "USER_JWT_KEY is not set") {
		t.Errorf("error %q should mention USER_JWT_KEY is not set", err)
	}
}

func TestInitUserJwtFromEnvParsesTTL(t *testing.T) {
	restoreUserJwt(t)
	t.Setenv("USER_JWT_KEY", "prod-style-user-jwt-key-32-bytes!!")
	t.Setenv("USER_JWT_TTL", "90s")
	t.Setenv("GCLOUD_PROJECT", "trustedassertions")

	if err := InitUserJwtFromEnv(); err != nil {
		t.Fatalf("InitUserJwtFromEnv: %v", err)
	}
	if string(userJwtKey) != "prod-style-user-jwt-key-32-bytes!!" {
		t.Errorf("userJwtKey = %q", userJwtKey)
	}
	if UserJwtTTL() != 90*time.Second {
		t.Errorf("TTL = %v, want 90s", UserJwtTTL())
	}
}

func TestInitUserJwtFromEnvRejectsInvalidTTL(t *testing.T) {
	t.Setenv("USER_JWT_KEY", "ok")
	t.Setenv("USER_JWT_TTL", "not-a-duration")
	t.Setenv("GCLOUD_PROJECT", "")

	if err := InitUserJwtFromEnv(); err == nil {
		t.Fatal("expected error for invalid USER_JWT_TTL")
	}
}

func TestInitUserJwtRejectsEmptyKey(t *testing.T) {
	if err := InitUserJwt("", time.Hour); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestMakeUserJwtRequiresConfiguredKey(t *testing.T) {
	restoreUserJwt(t)
	userJwtKey = nil

	token, err := MakeUserJwt("Tester")
	if err == nil {
		t.Fatal("expected error when user JWT key is not configured")
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
	if err != ErrUserJwtKeyNotSet {
		t.Errorf("error = %v, want ErrUserJwtKeyNotSet", err)
	}
}

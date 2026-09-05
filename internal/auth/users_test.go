package auth

import (
	"bytes"
	"encoding/json"
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

func TestRoles(t *testing.T) {
	user := User{Id: "x"}

	if user.HasRole(RoleAuthor) {
		t.Error("Expected user to have no roles initially")
	}

	user.AddRole(RoleAuthor)

	if !user.HasRole(RoleAuthor) {
		t.Error("Expected user to have Author after addition")
	}

	if user.HasRole(RoleAdministrator) {
		t.Error("Expected user to not have Administrator after adding Author")
	}

	user.AddRole(RoleAuthor)
	if len(user.Roles) != 1 {
		t.Errorf("Expected unique roles after duplicate add, got %v", user.Roles)
	}

	user.AddRole(RoleAdministrator)
	if !user.HasRole(RoleAdministrator) {
		t.Error("Expected user to have Administrator after addition")
	}
	if len(user.Roles) != 2 {
		t.Errorf("Expected two roles, got %v", user.Roles)
	}
}

func TestUserStatus(t *testing.T) {
	user := User{Id: "x"}
	if user.IsLocked() {
		t.Error("Expected user to be unlocked when status is empty")
	}

	user.SetStatus(UserStatusLocked)
	if !user.IsLocked() {
		t.Error("Expected user to be locked after SetStatus(Locked)")
	}
	if user.Status != UserStatusLocked {
		t.Errorf("Status = %q, want %q", user.Status, UserStatusLocked)
	}

	user.SetStatus(UserStatusActive)
	if user.IsLocked() {
		t.Error("Expected user to be unlocked after SetStatus(Active)")
	}
	if user.Status != UserStatusActive {
		t.Errorf("Status = %q, want %q", user.Status, UserStatusActive)
	}
}

func TestStatusFromJSON(t *testing.T) {
	var locked User
	if err := json.Unmarshal([]byte(`{"id":"alice","status":"Locked"}`), &locked); err != nil {
		t.Fatalf("unmarshal locked user: %v", err)
	}
	if !locked.IsLocked() {
		t.Errorf("expected Locked from JSON, got %q", locked.Status)
	}

	var unlocked User
	if err := json.Unmarshal([]byte(`{"id":"bob","passhash":"x"}`), &unlocked); err != nil {
		t.Fatalf("unmarshal user without status field: %v", err)
	}
	if unlocked.IsLocked() {
		t.Errorf("expected unlocked when status field is absent, got %q", unlocked.Status)
	}
}

func TestRolesFromJSON(t *testing.T) {
	var withRoles User
	if err := json.Unmarshal([]byte(`{"id":"alice","roles":["Author","Administrator"]}`), &withRoles); err != nil {
		t.Fatalf("unmarshal user with roles: %v", err)
	}
	if !withRoles.HasRole(RoleAuthor) || !withRoles.HasRole(RoleAdministrator) {
		t.Errorf("expected Author and Administrator from JSON, got %v", withRoles.Roles)
	}

	var withoutRoles User
	if err := json.Unmarshal([]byte(`{"id":"bob","passhash":"x"}`), &withoutRoles); err != nil {
		t.Fatalf("unmarshal user without roles: %v", err)
	}
	if withoutRoles.HasRole(RoleAuthor) || withoutRoles.HasRole(RoleAdministrator) {
		t.Errorf("expected no roles from JSON without roles field, got %v", withoutRoles.Roles)
	}
}

func TestParseBadJwt(t *testing.T) {
	jwt, err := parseUserJwt("broken", []byte("badkey"))

	if err == nil {
		t.Error("Parsing broken JWT did not return an error")
	}
	if jwt != "" {
		t.Error("Parsing broken JWT did not return an empty string")
	}
}

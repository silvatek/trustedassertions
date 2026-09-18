package auth

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/logging"
	log "silvatek.uk/trustedassertions/internal/logging"
	refs "silvatek.uk/trustedassertions/internal/references"
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

func TestAddTrustRoot(t *testing.T) {
	user := User{Id: "x"}
	entity := refs.UriFromString("hash://sha256/abc123")

	if user.HasTrustRoot(entity) {
		t.Error("Expected user to have no trust roots initially")
	}

	user.AddTrustRoot(entity, 0.50)

	if !user.HasTrustRoot(entity) {
		t.Error("Expected user to trust entity after add")
	}
	if len(user.TrustRoots) != 1 {
		t.Fatalf("TrustRoots len = %d, want 1", len(user.TrustRoots))
	}
	if user.TrustRoots[0].EntityUri != entity.Unadorned() {
		t.Errorf("EntityUri = %q, want %q", user.TrustRoots[0].EntityUri, entity.Unadorned())
	}
	if user.TrustRoots[0].TrustLevel != 0.50 {
		t.Errorf("TrustLevel = %v, want 0.50", user.TrustRoots[0].TrustLevel)
	}

	user.AddTrustRoot(entity.WithType("entity"), 0.90)
	if len(user.TrustRoots) != 1 {
		t.Errorf("expected no replace on duplicate entity, got %v", user.TrustRoots)
	}
	if user.TrustRoots[0].TrustLevel != 0.50 {
		t.Errorf("duplicate add replaced TrustLevel: %v", user.TrustRoots[0].TrustLevel)
	}
}

func TestAddTrustRootClampsLevel(t *testing.T) {
	user := User{Id: "x"}

	user.AddTrustRoot(refs.UriFromString("hash://sha256/low"), -0.20)
	if user.TrustRoots[0].TrustLevel != 0 {
		t.Errorf("negative TrustLevel = %v, want 0", user.TrustRoots[0].TrustLevel)
	}

	user.AddTrustRoot(refs.UriFromString("hash://sha256/high"), 1.5)
	if got := user.TrustRoots[1].TrustLevel; got != 1 {
		t.Errorf("over-one TrustLevel = %v, want 1", got)
	}

	user.AddTrustRoot(refs.UriFromString("hash://sha256/mid"), 0.21)
	if got := user.TrustRoots[2].TrustLevel; got != 0.21 {
		t.Errorf("in-range TrustLevel = %v, want 0.21", got)
	}
}

func TestAddTrustRootRejectsEmptyEntity(t *testing.T) {
	user := User{Id: "x"}
	user.AddTrustRoot(refs.EMPTY_URI, 0.50)
	if len(user.TrustRoots) != 0 {
		t.Errorf("expected empty entity to be rejected, got %v", user.TrustRoots)
	}
}

func TestTrustRootsFromJSON(t *testing.T) {
	var withRoots User
	if err := json.Unmarshal([]byte(`{"id":"alice","trust_roots":[{"entity_uri":"hash://sha256/abc","trust_level":0.5}]}`), &withRoots); err != nil {
		t.Fatalf("unmarshal user with trust_roots: %v", err)
	}
	if !withRoots.HasTrustRoot(refs.UriFromString("hash://sha256/abc")) {
		t.Errorf("expected trust root from JSON, got %v", withRoots.TrustRoots)
	}
	if withRoots.TrustRoots[0].TrustLevel != 0.50 {
		t.Errorf("TrustLevel from JSON = %v, want 0.50", withRoots.TrustRoots[0].TrustLevel)
	}

	var withoutRoots User
	if err := json.Unmarshal([]byte(`{"id":"bob","passhash":"x"}`), &withoutRoots); err != nil {
		t.Fatalf("unmarshal user without trust_roots: %v", err)
	}
	if len(withoutRoots.TrustRoots) != 0 {
		t.Errorf("expected no trust roots from JSON without field, got %v", withoutRoots.TrustRoots)
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

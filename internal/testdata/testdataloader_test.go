package testdata

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestInitialInviteRolesInMemory(t *testing.T) {
	roles := initialInviteRoles(datastore.NewInMemoryDataStore())
	if !containsRole(roles, auth.RoleAuthor) || !containsRole(roles, auth.RoleAdministrator) {
		t.Errorf("in-memory invite roles = %v, want Author and Administrator", roles)
	}
}

func TestInitialInviteRolesFirestore(t *testing.T) {
	roles := initialInviteRoles(&datastore.FireStore{})
	if roles != nil {
		t.Errorf("firestore invite roles = %v, want nil", roles)
	}
}

func containsRole(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}

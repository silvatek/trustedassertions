package web

import (
	"context"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestSuccessfulRegistration(t *testing.T) {
	ctx := context.Background()

	store := datastore.NewInMemoryDataStore()
	store.StoreRegistration(ctx, auth.Registration{Code: "abc", Status: "Pending"})

	regForm := RegistrationForm{
		regCode:   "ABC",
		userId:    "tester1",
		password1: "klsds877ds,wbdsfujehnsdcvbd£cioe",
		password2: "klsds877ds,wbdsfujehnsdcvbd£cioe",
	}

	regErr := registerUser(ctx, regForm, store)
	if regErr != nil {
		t.Errorf("Registration error: %v", regErr)
	}

	user, err := store.FetchUser(ctx, "tester1")
	if err != nil {
		t.Errorf("Error fetching new user: %v", regErr)
	}
	if len(user.Roles) != 0 {
		t.Errorf("expected no roles for invite without Roles, got %v", user.Roles)
	}

	reg, err := store.FetchRegistration(ctx, "abc")
	if err != nil {
		t.Errorf("Error fetching registration: %v", err)
	}
	if reg.Status != "Complete" {
		t.Errorf("Registration not marked as complete: %s", reg.Status)
	}
}

func TestRegistrationCopiesInviteRoles(t *testing.T) {
	ctx := context.Background()

	store := datastore.NewInMemoryDataStore()
	code := auth.NormalizeInviteCode("oak tree blue sky")
	store.StoreRegistration(ctx, auth.Registration{
		Code:   code,
		Status: "Pending",
		Roles:  []string{auth.RoleAuthor},
	})

	regForm := RegistrationForm{
		regCode:   "Oak Tree Blue Sky",
		userId:    "author1",
		password1: "klsds877ds,wbdsfujehnsdcvbd£cioe",
		password2: "klsds877ds,wbdsfujehnsdcvbd£cioe",
	}

	if err := registerUser(ctx, regForm, store); err != nil {
		t.Fatalf("Registration error: %v", err)
	}

	user, err := store.FetchUser(ctx, "author1")
	if err != nil {
		t.Fatalf("Error fetching new user: %v", err)
	}
	if !user.HasRole(auth.RoleAuthor) {
		t.Errorf("expected Author role copied from invite, got %v", user.Roles)
	}
	if user.HasRole(auth.RoleAdministrator) {
		t.Errorf("did not expect Administrator role, got %v", user.Roles)
	}
}

func TestFailedRegistration(t *testing.T) {
	ctx := context.Background()

	store := datastore.NewInMemoryDataStore()
	store.StoreRegistration(ctx, auth.Registration{Code: "abc", Status: "Pending"})
	store.StoreRegistration(ctx, auth.Registration{Code: "ijk", Status: "Complete"})
	store.StoreUser(ctx, auth.User{Id: "Existing"})

	testCases := []struct {
		name string
		reg  RegistrationForm
		err  int
	}{
		{name: "no code", reg: RegistrationForm{}, err: ErrorRegCode.ErrorCode},
		{name: "bad code", reg: RegistrationForm{regCode: "XYZ"}, err: ErrorRegCode.ErrorCode},
		{name: "used code", reg: RegistrationForm{regCode: "IJK"}, err: ErrorRegCode.ErrorCode},
		{name: "short username", reg: RegistrationForm{regCode: "ABC", userId: "A"}, err: ErrorBadUsername.ErrorCode},
		{name: "bad username", reg: RegistrationForm{regCode: "ABC", userId: "Ab/cde"}, err: ErrorBadUsername.ErrorCode},
		{name: "existing user", reg: RegistrationForm{regCode: "ABC", userId: "Existing"}, err: ErrorUserExists.ErrorCode},
		{name: "password mismatch", reg: RegistrationForm{regCode: "ABC", userId: "Tester", password1: "A", password2: "B"}, err: ErrorPasswordMismatch.ErrorCode},
		{name: "weak password", reg: RegistrationForm{regCode: "ABC", userId: "Tester", password1: "Password", password2: "Password"}, err: ErrorWeakPassword.ErrorCode},
	}

	for _, cfg := range testCases {
		err := registerUser(ctx, cfg.reg, store)
		if err == nil {
			t.Errorf("Unexpected registration success for `%s`", cfg.name)
		} else if err.ErrorCode != cfg.err {
			t.Errorf("Unexpected error for `%s`: %d", cfg.name, err.ErrorCode)
		}
	}
}

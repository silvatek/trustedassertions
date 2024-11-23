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
	store.StoreRegistration(ctx, auth.Registration{Code: "ABC", Status: "Pending"})

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

	_, err := store.FetchUser(ctx, "tester1")
	if err != nil {
		t.Errorf("Error fetching new user: %v", regErr)
	}

	reg, err := store.FetchRegistration(ctx, "ABC")
	if err != nil {
		t.Errorf("Error fetching new user: %v", regErr)
	}
	if reg.Status != "Complete" {
		t.Errorf("Registration not marked as complete: %s", reg.Status)
	}
}

func TestFailedRegistration(t *testing.T) {
	ctx := context.Background()

	store := datastore.NewInMemoryDataStore()
	store.StoreRegistration(ctx, auth.Registration{Code: "ABC", Status: "Pending"})
	store.StoreRegistration(ctx, auth.Registration{Code: "IJK", Status: "Complete"})
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

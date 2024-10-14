package auth

import "testing"

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

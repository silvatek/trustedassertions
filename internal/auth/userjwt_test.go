package auth

import "testing"

func TestJwtRoundTrip(t *testing.T) {
	key := MakeJwtKey()

	jwt, err := MakeUserJwt("Tester", key)
	if err != nil {
		t.Errorf("Error making JWT: %v", err)
	}
	t.Logf("JWT = %s", jwt)

	username, err := ParseUserJwt(jwt, key)
	if err != nil {
		t.Errorf("Error parsing JWT: %v", err)
	}

	if username != "Tester" {
		t.Errorf("Unexpected username: %s", username)
	}
}

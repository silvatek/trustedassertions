package datastore

import "testing"

func TestKeyNotFoundError(t *testing.T) {
	err := KeyNotFoundError{}
	if err.Error() != "Key not found" {
		t.Errorf("Unexpected error text: %s", err.Error())
	}
}

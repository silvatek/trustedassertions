package datastore

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
)

func TestStoreFetchStatement(t *testing.T) {
	assertions.InitKeyPair()
	InitInMemoryDataStore()

	statement1 := assertions.NewStatement("testing")
	uri := statement1.Uri()
	t.Log(uri)

	DataStore.Store(&statement1)

	statement2 := DataStore.FetchStatement(uri)

	if statement2.Content() != statement1.Content() {
		t.Errorf("Mismatched content: %s", statement2.Content())
	}
}

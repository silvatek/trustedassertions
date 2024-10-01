package datastore

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
)

func TestStoreFetchStatement(t *testing.T) {
	InitInMemoryDataStore()

	statement1 := assertions.NewStatement("testing")
	uri := statement1.Uri()
	t.Log(uri)

	ActiveDataStore.Store(&statement1)

	statement2, _ := ActiveDataStore.FetchStatement(uri)

	if statement2.Content() != statement1.Content() {
		t.Errorf("Mismatched content: %s", statement2.Content())
	}
}

package datastore

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
)

func TestPutGetEntity(t *testing.T) {
	assertions.InitKeyPair()
	InitInMemoryDataStore()

	entity1 := &assertions.Entity{
		CommonName: "John Smith",
		PublicKey:  assertions.PublicKey.N.String(),
	}

	ds := DataStore
	ds.data = make(map[string]string)

	key := ds.StoreClaims(entity1)

	t.Log(key)

	entity2, err := ds.FetchEntity(key)
	if err != nil {
		t.Error(err)
	}

	t.Log(entity2.CommonName)

	if entity2.CommonName != entity1.CommonName {
		t.Errorf("Fetched entity name mismatch: %s", entity2.CommonName)
	}
}

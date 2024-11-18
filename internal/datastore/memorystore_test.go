package datastore

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/entities"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

func TestMetadata(t *testing.T) {
	InitInMemoryDataStore()

	if ActiveDataStore.Name() != "InMemoryDataStore" {
		t.Errorf("Unexpected datastore name: %s", ActiveDataStore.Name())
	}

	if ActiveDataStore.AutoInit() == false {
		t.Error("In-memory datastore not set to auto init")
	}
}

func TestStoreFetchStatement(t *testing.T) {
	InitInMemoryDataStore()
	ctx := context.Background()

	statement1 := statements.NewStatement("testing")
	uri := statement1.Uri()
	t.Log(uri)

	ActiveDataStore.Store(context.TODO(), statement1)

	statement2, _ := ActiveDataStore.FetchStatement(ctx, uri)

	if statement2.Content() != statement1.Content() {
		t.Errorf("Mismatched content: %s", statement2.Content())
	}
}

func TestStoreFetchEntity(t *testing.T) {
	InitInMemoryDataStore()
	ctx := context.Background()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	entity1 := entities.NewEntity("Test Entity", *big.NewInt(123456))
	entity1.MakeCertificate(privateKey)
	uri := entity1.Uri()

	ActiveDataStore.Store(context.TODO(), &entity1)

	entity2, err := ActiveDataStore.FetchEntity(ctx, uri)
	if err != nil {
		t.Errorf("Unable to fetch new entity: %v", err)
	}
	if entity2.CommonName != entity1.CommonName {
		t.Errorf("Fetched entity has wrong name: %s", entity2.CommonName)
	}
}

func TestSearch(t *testing.T) {
	InitInMemoryDataStore()

	s := statements.NewStatement("Red Green Blue")
	ActiveDataStore.Store(context.TODO(), s)
	s = statements.NewStatement("Red Yellow Blue")
	ActiveDataStore.Store(context.TODO(), s)
	s = statements.NewStatement("White Green Blue")
	ActiveDataStore.Store(context.TODO(), s)

	matches, err := ActiveDataStore.Search("green")
	if err != nil {
		t.Errorf("Error fetching search results: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Unexpected number of search matches: %d", len(matches))
	}
}

func TestStoreFetchUser(t *testing.T) {
	InitInMemoryDataStore()

	user1 := auth.User{Id: "Tester", PassHash: "zzz"}
	user1.AddKeyRef("123", "Testing")
	ActiveDataStore.StoreUser(user1)

	user2, err := ActiveDataStore.FetchUser("Tester")
	if err != nil {
		t.Errorf("Error fetching new user: %v", err)
	}
	if !user2.HasKey("123") {
		t.Errorf("Fetched user does not have expected keyref")
	}
}

func TestStoreFetchReference(t *testing.T) {
	InitInMemoryDataStore()

	ref := Reference{
		Source:  MakeUri("123456", "assertion"),
		Target:  MakeUri("234567", "statement"),
		Summary: "Testing",
	}

	ActiveDataStore.StoreRef(context.TODO(), ref)

	refs, err := ActiveDataStore.FetchRefs(context.TODO(), ref.Target)
	if err != nil {
		t.Errorf("Error fetching references: %v", err)
	}
	if len(refs) != 1 {
		t.Errorf("Unexpected number of references: %d", len(refs))
	}
}

func TestStoreFetchKey(t *testing.T) {
	InitInMemoryDataStore()

	uri := MakeUri("123456", "entity")
	ActiveDataStore.StoreKey(uri, "kjsdfhfdksjhfdsjk")

	key, err := ActiveDataStore.FetchKey(uri)
	if err != nil {
		t.Errorf("Error fetching key: %v", err)
	}
	if key != "kjsdfhfdksjhfdsjk" {
		t.Errorf("Unexpected value of fetched key: %s", key)
	}
}

func storeStatements(content ...string) []HashUri {
	uris := make([]HashUri, len(content))
	for n, text := range content {
		statement := statements.NewStatement(text)
		uris[n] = statement.Uri()
		ActiveDataStore.Store(context.TODO(), statement)
	}
	return uris
}

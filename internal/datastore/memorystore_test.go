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

	matches, err := ActiveDataStore.Search(context.Background(), "green")
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
	ActiveDataStore.StoreUser(context.TODO(), user1)

	user2, err := ActiveDataStore.FetchUser(context.TODO(), "Tester")
	if err != nil {
		t.Errorf("Error fetching new user: %v", err)
	}
	if !user2.HasKey("123") {
		t.Errorf("Fetched user does not have expected keyref")
	}
}

func TestFetchUserCopiesRoles(t *testing.T) {
	InitInMemoryDataStore()
	ctx := context.TODO()

	user1 := auth.User{Id: "Tester"}
	user1.AddRole(auth.RoleAuthor)
	ActiveDataStore.StoreUser(ctx, user1)

	user2, err := ActiveDataStore.FetchUser(ctx, "Tester")
	if err != nil {
		t.Fatalf("Error fetching user: %v", err)
	}
	if !user2.HasRole(auth.RoleAuthor) {
		t.Errorf("Fetched user missing Author role")
	}

	user2.AddRole(auth.RoleAdministrator)

	user3, err := ActiveDataStore.FetchUser(ctx, "Tester")
	if err != nil {
		t.Fatalf("Error refetching user: %v", err)
	}
	if user3.HasRole(auth.RoleAdministrator) {
		t.Errorf("AddRole on fetched user mutated stored roles: %v", user3.Roles)
	}
}

func TestStoreFetchRegistrationCopiesRoles(t *testing.T) {
	InitInMemoryDataStore()
	ctx := context.TODO()

	code := auth.NormalizeInviteCode("oak tree blue sky")
	ActiveDataStore.StoreRegistration(ctx, auth.Registration{
		Code:   code,
		Status: "Pending",
		Roles:  []string{auth.RoleAuthor},
	})

	reg, err := ActiveDataStore.FetchRegistration(ctx, code)
	if err != nil {
		t.Fatalf("FetchRegistration: %v", err)
	}
	if !containsRole(reg.Roles, auth.RoleAuthor) {
		t.Errorf("fetched registration missing Author role: %v", reg.Roles)
	}

	reg.Roles = append(reg.Roles, auth.RoleAdministrator)

	again, err := ActiveDataStore.FetchRegistration(ctx, code)
	if err != nil {
		t.Fatalf("refetch registration: %v", err)
	}
	if containsRole(again.Roles, auth.RoleAdministrator) {
		t.Errorf("mutating fetched registration roles mutated store: %v", again.Roles)
	}
}

func TestListRegistrations(t *testing.T) {
	InitInMemoryDataStore()
	ctx := context.TODO()

	ActiveDataStore.StoreRegistration(ctx, auth.Registration{
		Code:   "oaktreebluesky",
		Status: "Pending",
		Roles:  []string{auth.RoleAuthor},
	})
	ActiveDataStore.StoreRegistration(ctx, auth.Registration{
		Code:   "riverstonehillpath",
		Status: "Complete",
	})

	regs, err := ActiveDataStore.ListRegistrations(ctx)
	if err != nil {
		t.Fatalf("ListRegistrations: %v", err)
	}
	if len(regs) != 2 {
		t.Fatalf("expected 2 registrations, got %d", len(regs))
	}

	byCode := make(map[string]auth.Registration, len(regs))
	for _, reg := range regs {
		byCode[reg.Code] = reg
	}
	pending := byCode["oaktreebluesky"]
	if pending.Status != "Pending" {
		t.Errorf("pending status: %s", pending.Status)
	}
	if !containsRole(pending.Roles, auth.RoleAuthor) {
		t.Errorf("pending missing Author: %v", pending.Roles)
	}

	pending.Roles = append(pending.Roles, auth.RoleAdministrator)
	listed, err := ActiveDataStore.ListRegistrations(ctx)
	if err != nil {
		t.Fatalf("ListRegistrations again: %v", err)
	}
	for _, reg := range listed {
		if reg.Code == "oaktreebluesky" && containsRole(reg.Roles, auth.RoleAdministrator) {
			t.Errorf("mutating listed registration roles mutated store: %v", reg.Roles)
		}
	}
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
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

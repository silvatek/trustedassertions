package datastore

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/statements"
)

func TestContentMatches(t *testing.T) {
	data := map[[2]string]bool{
		{"red", "blue red green"}:   true,
		{"white", "blue red green"}: false,
		{"red", ""}:                 false,
	}
	for key, expected := range data {
		if contentMatches(key[1], key[0]) != expected {
			t.Errorf("Unexpected match: %s, %s, %v", key[0], key[1], expected)
		}
	}
}

func TestDataMap(t *testing.T) {
	statement := statements.NewStatement("Testing")

	data := contentDataMap(statement)

	if !strings.HasPrefix(data["uri"].(string), "hash://sha256/") {
		t.Errorf("Did not map URI as expected: %s", data["uri"])
	}
	if data["content"] != "Testing" {
		t.Errorf("Did not map content as expected: %s", data["content"])
	}

}

func TestFirestoreSearch(t *testing.T) {
	df := DocFetcher{testData: []DbRecord{
		{Uri: "1", Content: "Red", Summary: "Red", DataType: "Statement"},
		{Uri: "2", Content: "Blue Red", DataType: "Statement"},
		{Uri: "3", Summary: "Green Blue"},
		{Uri: "4", Summary: "Mr Red", DataType: "Entity"},
		{Uri: "5", Content: "Red Assertion", DataType: "Assertion"},
		{Uri: "6", Summary: "Green Red"},
	}}

	matches := searchDocs(df, "red")

	if len(matches) != 4 {
		t.Errorf("Unexpected number of matches: %d", len(matches))
	}
}

func TestFireStoreUserPasskeyDocumentRoundTrip(t *testing.T) {
	user := auth.User{Id: "alice", PassHash: "hash"}
	pk := auth.Passkey{
		ID:         []byte{0xff, 0x00, 0xab},
		PublicKey:  []byte{1, 2, 3},
		SignCount:  4,
		Name:       "Phone",
		CreatedAt:  time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		LastUsedAt: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC),
		Transports: []string{"internal"},
	}
	if err := user.AddPasskey(pk); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user document: %v", err)
	}
	var got auth.User
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal user document: %v", err)
	}

	if len(got.Passkeys) != 1 {
		t.Fatalf("passkeys = %d", len(got.Passkeys))
	}
	if !bytes.Equal(got.Passkeys[0].ID, pk.ID) || !bytes.Equal(got.Passkeys[0].PublicKey, pk.PublicKey) {
		t.Error("binary passkey fields did not survive document encoding")
	}
	if got.Passkeys[0].SignCount != 4 || got.Passkeys[0].Name != "Phone" {
		t.Errorf("passkey = %+v", got.Passkeys[0])
	}
}

package datastore

import (
	"strings"
	"testing"

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

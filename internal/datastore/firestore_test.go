package datastore

import (
	"testing"
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

func TestFirestoreSearch(t *testing.T) {
	df := DocFetcher{testData: []DbRecord{
		{Uri: "123", Content: "Red", Summary: "Red"},
	}}
	matches := doSearch(df, "red")
	if len(matches) != 1 {
		t.Errorf("Unexpected number of matches: %d", len(matches))
	}
}

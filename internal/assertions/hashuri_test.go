package assertions

import "testing"

func TestBasicHashUri(t *testing.T) {
	data := map[[2]string]string{
		{"12345678", "statement"}: "hash://sha256/12345678?type=statement",
		{"12345678", ""}:          "hash://sha256/12345678",
		{"", ""}:                  "hash://sha256/",
	}

	for input, expected := range data {
		hashUri := MakeUri(input[0], input[1])
		if hashUri.String() != expected {
			t.Errorf("Unexptected HashUri (%s, %s): %s", input[0], input[1], hashUri)
		}
		if hashUri.Alg() != "sha256" {
			t.Errorf("Unexpected algorithm: %s", hashUri.Alg())
		}
	}
}

func TestHashUriHash(t *testing.T) {
	data := map[string]string{
		"hash://sha256/12345678":             "12345678",
		"hash://sha256/12345678?type=entity": "12345678",
		"12345678":                           "12345678",
		"hash://sha256/?type=entity":         "",
		"":                                   "",
	}

	for input, expected := range data {
		uri := UriFromString(input)
		hash := uri.Hash()
		if hash != expected {
			t.Errorf("Unexpected hash for %s - %s", input, hash)
		}
	}
}

func TestHashShort(t *testing.T) {
	data := map[string]string{
		"hash://sha256/1234567890":             "34567890",
		"hash://sha256/1234567890?type=entity": "34567890",
		"1234567890":                           "34567890",
		"hash://sha256/?type=entity":           "",
		"":                                     "",
		"hash://sha256/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db?type=entity": "265df2db",
	}

	for input, expected := range data {
		uri := HashUri{uri: input}
		short := uri.Short()
		if short != expected {
			t.Errorf("Unexpected short hash for %s => %s", input, short)
		}
	}
}

func TestHashPath(t *testing.T) {
	data := map[string][2]string{
		"hash://sha256/12345678?type=entity": {"/web/entities/12345678", "/api/v1/entities/12345678"},
	}

	for input, expected := range data {
		uri := HashUri{uri: input}
		path := uri.WebPath()
		if path != expected[0] {
			t.Errorf("Unexpected web path for uri %s ==> %s", input, path)
		}
		path = uri.ApiPath()
		if path != expected[1] {
			t.Errorf("Unexpected api path for uri %s ==> %s", input, path)
		}
	}
}

func TestUnadornedUri(t *testing.T) {
	uri := MakeUri("12345678", "statement")
	if uri.Unadorned() != "hash://sha256/12345678" {
		t.Errorf("Incorrect unadorned URI: %s", uri.Unadorned())
	}
}

func TestEscaped(t *testing.T) {
	u := MakeUri("12345678", "statement")
	if u.Escaped() != "hash:%2F%2Fsha256%2F12345678" {
		t.Errorf("Unexpected escaped URI: %s", u.Escaped())
	}

	u1 := UnescapeUri(u.Escaped(), "statement")
	if u1.String() != "hash://sha256/12345678?type=statement" {
		t.Errorf("Unexpected round-tripped URI: %s", u1.String())
	}
}

func TestUriType(t *testing.T) {
	uri := MakeUri("12345678", "")
	if uri.HasType() {
		t.Error("Expected URI to not have type")
	}

	uri = uri.WithType("statement")
	if uri.Kind() != "statement" {
		t.Errorf("Unexpected URI type: %s", uri.Kind())
	}
}

func TestEquals(t *testing.T) {
	u1 := MakeUri("1234", "statement")
	u2 := MakeUri("1234", "")
	if !u1.Equals(u2) {
		t.Error("Expected URIs to be equal")
	}
}

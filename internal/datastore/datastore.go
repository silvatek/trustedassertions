package datastore

import (
	"silvatek.uk/trustedassertions/internal/assertions"
)

type DataStore interface {
	Name() string
	Store(value assertions.Referenceable)
	FetchStatement(key string) assertions.Statement
	FetchEntity(key string) assertions.Entity
	FetchAssertion(key string) assertions.Assertion
}

type KeyNotFoundError struct {
}

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

type InMemoryDataStore struct {
	data map[string]string
}

var ActiveDataStore DataStore

func InitInMemoryDataStore() {
	datastore := InMemoryDataStore{}
	datastore.data = make(map[string]string)
	ActiveDataStore = &datastore
}

func (ds *InMemoryDataStore) Name() string {
	return "InMemoryDataStore"
}

func (ds *InMemoryDataStore) Store(value assertions.Referenceable) {
	ds.data[value.Uri()] = value.Content()
}

func (ds *InMemoryDataStore) FetchStatement(key string) assertions.Statement {
	content := ds.data[key]
	return assertions.NewStatement(content)
}

func (ds *InMemoryDataStore) FetchEntity(key string) assertions.Entity {
	content := ds.data[key]
	return assertions.ParseCertificate(content)
}

func (ds *InMemoryDataStore) FetchAssertion(key string) assertions.Assertion {
	content := ds.data[key]
	assertion, _ := assertions.ParseAssertionJwt(content)
	return assertion
}

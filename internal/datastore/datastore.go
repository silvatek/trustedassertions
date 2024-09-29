package datastore

import (
	"errors"
	"log"

	"silvatek.uk/trustedassertions/internal/assertions"
)

type DataStore interface {
	Name() string
	Store(value assertions.Referenceable)
	StoreRaw(uri string, content string)
	FetchStatement(key string) (assertions.Statement, error)
	FetchEntity(key string) (assertions.Entity, error)
	FetchAssertion(key string) (assertions.Assertion, error)
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

func (ds *InMemoryDataStore) StoreRaw(uri string, content string) {
	log.Printf("Storing %s", uri)
	ds.data[uri] = content
}

func (ds *InMemoryDataStore) Store(value assertions.Referenceable) {
	ds.StoreRaw(value.Uri(), value.Content())
}

func (ds *InMemoryDataStore) FetchStatement(key string) (assertions.Statement, error) {
	content, ok := ds.data[key]
	if !ok {
		return assertions.Statement{}, errors.New("statement not found: " + key)
	}
	return assertions.NewStatement(content), nil
}

func (ds *InMemoryDataStore) FetchEntity(key string) (assertions.Entity, error) {
	content := ds.data[key]
	return assertions.ParseCertificate(content), nil
}

func (ds *InMemoryDataStore) FetchAssertion(key string) (assertions.Assertion, error) {
	content := ds.data[key]
	assertion, err := assertions.ParseAssertionJwt(content)
	if err != nil {
		log.Printf("Error parsing assertion JWT: %v", err)
	}
	return assertion, err
}

package datastore

import (
	"errors"

	"silvatek.uk/trustedassertions/internal/assertions"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type DataStore interface {
	Name() string
	Store(value assertions.Referenceable)
	StoreRaw(uri string, content string)
	StoreKey(entityUri string, key string)
	FetchStatement(key string) (assertions.Statement, error)
	FetchEntity(key string) (assertions.Entity, error)
	FetchAssertion(key string) (assertions.Assertion, error)
	FetchKey(entityUri string) (string, error)
}

type KeyNotFoundError struct {
}

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

type InMemoryDataStore struct {
	data map[string]string
	keys map[string]string
}

var ActiveDataStore DataStore

func InitInMemoryDataStore() {
	datastore := InMemoryDataStore{}
	datastore.data = make(map[string]string)
	datastore.keys = make(map[string]string)
	ActiveDataStore = &datastore
}

func (ds *InMemoryDataStore) Name() string {
	return "InMemoryDataStore"
}

func (ds *InMemoryDataStore) StoreRaw(uri string, content string) {
	log.Debugf("Storing %s", uri)
	ds.data[uri] = content
}

func (ds *InMemoryDataStore) Store(value assertions.Referenceable) {
	ds.StoreRaw(value.Uri(), value.Content())
}

func (ds *InMemoryDataStore) StoreKey(entityUri string, key string) {
	ds.keys[entityUri] = key
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
		log.Errorf("Error parsing assertion JWT: %v", err)
	}
	return assertion, err
}

func (ds *InMemoryDataStore) FetchKey(entityUri string) (string, error) {
	key, ok := ds.keys[entityUri]
	if !ok {
		return "", errors.New("entity id not found " + entityUri)
	}
	return key, nil
}

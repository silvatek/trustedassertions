package datastore

import (
	"silvatek.uk/trustedassertions/internal/assertions"
)

// type DataStore interface {
// 	Store(key string, value string)
// 	Fetch(key string) (string, error)
// 	FetchEntity(key string) (assertions.Entity, error)
// }

type KeyNotFoundError struct {
}

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

type InMemoryDataStore struct {
	data map[string]string
}

var DataStore InMemoryDataStore

func InitInMemoryDataStore() {
	DataStore = InMemoryDataStore{}
	DataStore.data = make(map[string]string)
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

// func (ds *InMemoryDataStore) FetchEntity(key string) (assertions.Entity, error) {
// 	value, err := ds.Fetch(key)
// 	if err != nil {
// 		return assertions.Entity{}, err
// 	}
// 	token := key + "." + value
// 	entity, err := assertions.ParseEntityJwt(token)
// 	return entity, err
// }

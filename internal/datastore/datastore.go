package datastore

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
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

func (ds *InMemoryDataStore) Store(key string, value string) {
	ds.data[key] = value
}

func (ds *InMemoryDataStore) Fetch(key string) (string, error) {
	value, ok := ds.data[key]
	if !ok {
		return "", &KeyNotFoundError{}
	}
	return value, nil
}

func (ds *InMemoryDataStore) StoreClaims(value jwt.Claims) string {
	token, _ := assertions.CreateJwt(value)
	parts := strings.SplitN(token, ".", 2)
	ds.Store(parts[0], parts[1])
	return parts[0]
}

func (ds *InMemoryDataStore) FetchEntity(key string) (assertions.Entity, error) {
	value, err := ds.Fetch(key)
	if err != nil {
		return assertions.Entity{}, err
	}
	token := key + "." + value
	entity, err := assertions.ParseEntityJwt(token)
	return entity, err
}

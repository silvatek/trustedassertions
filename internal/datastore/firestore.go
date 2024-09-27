package datastore

import (
	"log"
	"math/big"

	"silvatek.uk/trustedassertions/internal/assertions"
)

type FireStore struct {
}

func InitFireStore() {
	log.Print("Initialising FireStore")
	ActiveDataStore = &FireStore{}
}

func (fs *FireStore) Name() string {
	return "FireStore"
}

func (fs *FireStore) Store(value assertions.Referenceable) {}

func (fs *FireStore) FetchStatement(key string) assertions.Statement {
	return assertions.NewStatement("{empty}")
}

func (fs *FireStore) FetchEntity(key string) assertions.Entity {
	return assertions.NewEntity("{empty}", *big.NewInt(0))
}

func (fs *FireStore) FetchAssertion(key string) assertions.Assertion {
	return assertions.NewAssertion("{empty}")
}

package datastore

import (
	"context"
	"log"
	"math/big"
	"os"

	"cloud.google.com/go/firestore"
	"silvatek.uk/trustedassertions/internal/assertions"
)

type FireStore struct {
	projectId    string
	databaseName string
}

func InitFireStore() {
	datastore := &FireStore{
		projectId:    os.Getenv("GCLOUD_PROJECT"),
		databaseName: os.Getenv("trustedassertions"),
	}
	log.Printf("Initialising FireStore: %s / %s", datastore.projectId, datastore.databaseName)
	ActiveDataStore = datastore
}

func (fs *FireStore) Name() string {
	return "FireStore"
}

func (fs *FireStore) Store(value assertions.Referenceable) {}

func (fs *FireStore) FetchStatement(key string) assertions.Statement {
	return assertions.NewStatement("{empty}")
}

func (fs *FireStore) FetchEntity(key string) assertions.Entity {
	projectID := os.Getenv("GCLOUD_PROJECT")
	database := os.Getenv("trustedassertions")
	client, _ := firestore.NewClientWithDatabase(context.Background(), projectID, database)
	log.Print(client)
	return assertions.NewEntity("{empty}", *big.NewInt(0))
}

func (fs *FireStore) FetchAssertion(key string) assertions.Assertion {
	return assertions.NewAssertion("{empty}")
}

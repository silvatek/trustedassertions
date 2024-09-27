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
		databaseName: os.Getenv("FIRESTORE_DB_NAME"),
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
	log.Printf("Fetch entity %s", key)
	ctx := context.TODO()
	client, err := firestore.NewClientWithDatabase(ctx, fs.projectId, fs.databaseName)
	if err == nil {
		log.Print(err)
	} else {
		log.Print(client.Collections(ctx).GetAll())
	}
	return assertions.NewEntity("{empty}", *big.NewInt(0))
}

func (fs *FireStore) FetchAssertion(key string) assertions.Assertion {
	return assertions.NewAssertion("{empty}")
}

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

const MainCollection = "Primary"

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

func (fs *FireStore) client(ctx context.Context) *firestore.Client {
	client, err := firestore.NewClientWithDatabase(ctx, fs.projectId, fs.databaseName)
	if err == nil {
		log.Printf("Error connecting to database: %v", err)
	} else {
		log.Printf("Connected to Firestore database: %v", client)
	}
	return client
}

func (fs *FireStore) Store(value assertions.Referenceable) {
	ctx := context.TODO()
	log.Printf("Writing to datastore: %s", value.Uri())
	result, err := fs.client(ctx).Collection(MainCollection).Doc(value.Uri()).Set(ctx, value.Content())
	if err != nil {
		log.Printf("Error writing value: %v", err)
	} else {
		log.Printf("Written: %v", result)
	}
}

func (fs *FireStore) FetchStatement(key string) assertions.Statement {
	return assertions.NewStatement("{empty}")
}

func (fs *FireStore) FetchEntity(key string) assertions.Entity {
	log.Printf("Fetch entity %s", key)
	ctx := context.TODO()
	doc := fs.client(ctx).Collection("Test1").Doc("zLvK61myGkmekzGwsO7Z")
	data, err := doc.Get(context.TODO())
	if err != nil {
		log.Printf("Error getting doc: %v", err)
	} else {
		log.Printf("Retrieved doc: %v", data.Data())
	}
	return assertions.NewEntity("{empty}", *big.NewInt(0))
}

func (fs *FireStore) FetchAssertion(key string) assertions.Assertion {
	return assertions.NewAssertion("{empty}")
}

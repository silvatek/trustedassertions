package datastore

import (
	"context"
	"log"
	"math/big"
	"os"
	"strings"

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
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
	} else {
		log.Printf("Connected to Firestore database: %v", client)
	}
	return client
}

func (fs *FireStore) Store(value assertions.Referenceable) {
	log.Printf("Writing to datastore: %s", value.Uri())

	data := make(map[string]string)
	data["uri"] = value.Uri()
	data["content"] = value.Content()

	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	result, err := client.Collection(MainCollection).Doc(safeKey(value.Uri())).Set(ctx, data)

	if err != nil {
		log.Printf("Error writing value: %v", err)
	} else {
		log.Printf("Written: %v", result)
	}
}

func safeKey(uri string) string {
	uri = strings.ReplaceAll(uri, ":", "_")
	uri = strings.ReplaceAll(uri, "/", "_")
	return uri
}

type DbRecord struct {
	uri     string
	content string
}

func (fs *FireStore) fetch(key string) (*DbRecord, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	data, err := client.Collection(MainCollection).Doc(safeKey(key)).Get(ctx)
	if err != nil {
		log.Printf("Error reading value: %v", err)
		return nil, err
	} else {
		record := DbRecord{}
		data.DataTo(&record)
		return &record, nil
	}
}

func (fs *FireStore) FetchStatement(key string) assertions.Statement {
	record, err := fs.fetch(key)

	if err != nil {
		return assertions.NewStatement("")
	} else {
		return assertions.NewStatement(record.content)
	}
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

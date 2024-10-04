package datastore

import (
	"context"
	"math/big"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"silvatek.uk/trustedassertions/internal/assertions"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type FireStore struct {
	projectId    string
	databaseName string
}

const MainCollection = "Primary"
const KeyCollection = "Keys"

func InitFireStore() {
	datastore := &FireStore{
		projectId:    os.Getenv("GCLOUD_PROJECT"),
		databaseName: os.Getenv("FIRESTORE_DB_NAME"),
	}
	log.Infof("Initialising FireStore: %s / %s", datastore.projectId, datastore.databaseName)
	ActiveDataStore = datastore
}

func (fs *FireStore) Name() string {
	return "FireStore"
}

func (fs *FireStore) client(ctx context.Context) *firestore.Client {
	client, err := firestore.NewClientWithDatabase(ctx, fs.projectId, fs.databaseName)
	if err != nil {
		log.Errorf("Error connecting to database: %v", err)
	} else {
		log.Debugf("Connected to Firestore database: %v", client)
	}
	return client
}

func (fs *FireStore) StoreRaw(uri assertions.HashUri, content string) {
	log.Debugf("Writing to datastore: %s", uri)

	data := make(map[string]string)
	data["uri"] = uri.Unadorned()
	data["content"] = content
	data["datatype"] = "unknown"

	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	result, err := client.Collection(MainCollection).Doc(safeKey(uri.Unadorned())).Set(ctx, data)

	if err != nil {
		log.Errorf("Error writing value: %v", err)
	} else {
		log.Debugf("Written: %v", result)
	}
}

func (fs *FireStore) Store(value assertions.Referenceable) {
	log.Debugf("Writing to datastore: %s", value.Uri())

	data := make(map[string]string)
	data["uri"] = value.Uri().Unadorned()
	data["content"] = value.Content()
	data["datatype"] = value.Type()

	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	result, err := client.Collection(MainCollection).Doc(safeKey(value.Uri().Unadorned())).Set(ctx, data)

	if err != nil {
		log.Errorf("Error writing value: %v", err)
	} else {
		log.Debugf("Written: %v", result)
	}
}

func (fs *FireStore) StoreKey(entityUri assertions.HashUri, key string) {
	data := make(map[string]string)
	data["entity"] = entityUri.Unadorned()
	data["encoding"] = "base64"
	data["key"] = key

	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	result, err := client.Collection(KeyCollection).Doc(safeKey(entityUri.Unadorned())).Set(ctx, data)

	if err != nil {
		log.Errorf("Error writing key: %v", err)
	} else {
		log.Debugf("Written: %v", result)
	}
}

func safeKey(uri string) string {
	index := strings.Index(uri, "?type=")
	if index > -1 {
		uri = uri[0:index]
	}
	uri = strings.ReplaceAll(uri, ":", "_")
	uri = strings.ReplaceAll(uri, "/", "_")
	return uri
}

type DbRecord struct {
	Uri      string `json:"uri"`
	Content  string `json:"content"`
	DataType string `json:"datatype"`
	Summary  string `json:"summary"`
}

func (fs *FireStore) fetch(key string) (*DbRecord, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	doc, err := client.Collection(MainCollection).Doc(safeKey(key)).Get(ctx)
	if err != nil {
		log.Errorf("Error reading value: %v", err)
		return nil, err
	} else {
		record := DbRecord{}
		doc.DataTo(&record)
		return &record, nil
	}
}

func (fs *FireStore) FetchStatement(key string) (assertions.Statement, error) {
	record, err := fs.fetch(key)

	if err != nil {
		return assertions.NewStatement("{bad record}"), err
	} else {
		return assertions.NewStatement(record.Content), nil
	}
}

func (fs *FireStore) FetchEntity(key string) (assertions.Entity, error) {
	record, err := fs.fetch(key)

	if err != nil {
		return assertions.NewEntity("{bad record}", *big.NewInt(0)), err
	} else {
		return assertions.ParseCertificate(record.Content), nil
	}
}

func (fs *FireStore) FetchAssertion(key string) (assertions.Assertion, error) {
	record, err := fs.fetch(key)

	if err != nil {
		return assertions.NewAssertion("{bad record}"), err
	}

	assertion, err := assertions.ParseAssertionJwt(record.Content)
	if err != nil {
		log.Errorf("Error parsing JWT: %v", err)
		return assertions.NewAssertion("{bad record}"), err
	}
	return assertion, nil
}

type KeyRecord struct {
	Entity   string `json:"entity"`
	Key      string `json:"key"`
	Encoding string `json:"encoding"`
}

func (fs *FireStore) FetchKey(entityUri string) (string, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	doc, err := client.Collection(KeyCollection).Doc(safeKey(entityUri)).Get(ctx)
	if err != nil {
		log.Errorf("Error reading value: %v", err)
		return "", err
	} else {
		record := KeyRecord{}
		doc.DataTo(&record)
		return record.Key, nil
	}
}

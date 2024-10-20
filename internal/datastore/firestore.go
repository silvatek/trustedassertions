package datastore

import (
	"context"
	"math/big"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type FireStore struct {
	projectId    string
	databaseName string
}

const MainCollection = "Primary"
const KeyCollection = "Keys"
const UserCollection = "Users"

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

func (fs *FireStore) AutoInit() bool {
	return false
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

func (fs *FireStore) store(collection string, id string, data map[string]string) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	result, err := client.Collection(collection).Doc(id).Set(ctx, data)

	if err != nil {
		log.Errorf("Error writing value: %v", err)
	} else {
		log.Debugf("Written: %s %v", id, result)
	}
}

func rawDataMap(uri assertions.HashUri, content string, summary string) map[string]string {
	data := make(map[string]string)

	data["uri"] = uri.String()
	data["content"] = content
	data["datatype"] = uri.Kind()
	data["updated"] = time.Now().Format(time.RFC3339)

	if summary != "" {
		data["summary"] = summary
	}

	return data
}

func contentDataMap(value assertions.Referenceable) map[string]string {
	uri := value.Uri()
	if value.Type() != "" && !uri.HasType() {
		uri = uri.WithType(value.Type())
	}
	data := rawDataMap(uri, value.Content(), value.Summary())
	return data
}

func (fs *FireStore) StoreRaw(uri assertions.HashUri, content string) {
	log.Debugf("Writing to datastore: %s", uri)

	// data := make(map[string]string)
	// data["uri"] = uri.String()
	// data["content"] = content
	// data["datatype"] = uri.Kind()
	// data["updated"] = time.Now().Format(time.RFC3339)

	fs.store(MainCollection, uri.Escaped(), rawDataMap(uri, content, ""))
}

func (fs *FireStore) Store(value assertions.Referenceable) {
	log.Debugf("Writing to datastore: %s", value.Uri())

	// data := make(map[string]string)
	// data["uri"] = value.Uri().Unadorned()
	// data["content"] = value.Content()
	// data["datatype"] = value.Type()
	// data["summary"] = value.Summary()
	// data["updated"] = time.Now().Format(time.RFC3339)

	fs.store(MainCollection, value.Uri().Escaped(), contentDataMap(value))
}

func (fs *FireStore) StoreKey(entityUri assertions.HashUri, key string) {
	data := make(map[string]string)
	data["entity"] = entityUri.Unadorned()
	data["encoding"] = "base64"
	data["key"] = key

	fs.store(KeyCollection, entityUri.Escaped(), data)
}

func (fs *FireStore) StoreRef(source assertions.HashUri, target assertions.HashUri, refType string) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	data := make(map[string]string)
	data["source"] = source.String()
	data["target"] = target.String()
	data["type"] = refType

	refs := client.Collection(MainCollection).Doc(target.Escaped()).Collection("refs")
	refs.Doc(source.Escaped()).Set(ctx, data)
}

type DbRecord struct {
	Uri      string `json:"uri"`
	Content  string `json:"content"`
	DataType string `json:"datatype"`
	Summary  string `json:"summary"`
}

func (fs *FireStore) fetch(uri assertions.HashUri) (*DbRecord, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	doc, err := client.Collection(MainCollection).Doc(uri.Escaped()).Get(ctx)
	if err != nil {
		log.Errorf("Error reading value: %v", err)
		return nil, err
	} else {
		record := DbRecord{}
		doc.DataTo(&record)
		return &record, nil
	}
}

func (fs *FireStore) FetchStatement(uri assertions.HashUri) (assertions.Statement, error) {
	record, err := fs.fetch(uri)

	if err != nil {
		return assertions.NewStatement("{bad record}"), err
	} else {
		return assertions.NewStatement(record.Content), nil
	}
}

func (fs *FireStore) FetchEntity(uri assertions.HashUri) (assertions.Entity, error) {
	record, err := fs.fetch(uri)

	if err != nil {
		return assertions.NewEntity("{bad record}", *big.NewInt(0)), err
	} else {
		return assertions.ParseCertificate(record.Content), nil
	}
}

func (fs *FireStore) FetchAssertion(uri assertions.HashUri) (assertions.Assertion, error) {
	record, err := fs.fetch(uri)

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

func (fs *FireStore) FetchKey(entityUri assertions.HashUri) (string, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	doc, err := client.Collection(KeyCollection).Doc(entityUri.Escaped()).Get(ctx)
	if err != nil {
		log.Errorf("Error reading value: %v", err)
		return "", err
	} else {
		record := KeyRecord{}
		doc.DataTo(&record)
		return record.Key, nil
	}
}

func (fs *FireStore) FetchRefs(uri assertions.HashUri) ([]assertions.HashUri, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	results := make([]assertions.HashUri, 0)

	refs := client.Collection(MainCollection).Doc(uri.Escaped()).Collection("refs").Documents(ctx)
	for {
		doc, err := refs.Next()
		if err == iterator.Done {
			break
		}
		uri := assertions.UnescapeUri(doc.Ref.ID, "assertion")
		results = append(results, uri)
	}

	return results, nil
}

func (fs *FireStore) StoreUser(user auth.User) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	client.Collection(UserCollection).Doc(user.Id).Set(ctx, user)

	log.Debugf("Stored user %s", user.Id)
}

func (fs *FireStore) FetchUser(id string) (auth.User, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	user := auth.User{}

	doc, err := client.Collection(UserCollection).Doc(id).Get(ctx)
	if err != nil {
		return user, err
	}
	doc.DataTo(&user)

	return user, nil
}

// Thin wrapper around firestore.DocumentIterator that allows for mocking.
type DocFetcher struct {
	testData  []DbRecord
	testIndex int
	iterator  *firestore.DocumentIterator
}

func (df *DocFetcher) Next() *DbRecord {
	if df.testData != nil {
		if df.testIndex >= len(df.testData) {
			return nil
		}
		next := df.testData[df.testIndex]
		df.testIndex++
		return &next
	}

	doc, err := df.iterator.Next()
	if err == iterator.Done {
		return nil
	}
	record := DbRecord{}
	doc.DataTo(&record)
	return &record
}

func (fs *FireStore) Search(query string) ([]SearchResult, error) {
	ctx := context.TODO()
	client := fs.client(ctx)
	defer client.Close()

	df := DocFetcher{iterator: client.Collection(MainCollection).Documents(ctx)}

	results := searchDocs(df, query)

	return results, nil
}

func searchDocs(docs DocFetcher, query string) []SearchResult {
	results := make([]SearchResult, 0)
	for {
		record := docs.Next()
		if record == nil {
			break
		}

		if strings.ToLower(record.DataType) == "assertion" {
			// Don't bother searching assertions as they don't have textual content
			continue
		}

		summary := record.Summary
		if summary == "" && strings.ToLower(record.DataType) == "statement" {
			summary = record.Content
		} else if summary == "" && strings.ToLower(record.DataType) == "entity" {
			summary = assertions.ParseCertificate(record.Content).CommonName
		}
		log.Debugf("Searching %s => %s", record.Uri, summary)

		if contentMatches(summary, query) {
			uri := assertions.UriFromString(record.Uri)
			if !uri.HasType() {
				uri = uri.WithType(assertions.GuessContentType(record.Content))
			}

			result := SearchResult{
				Uri:       uri,
				Content:   summary,
				Relevance: 0.8,
			}

			results = append(results, result)
		}
	}
	return results
}

func contentMatches(content string, query string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(query))
}

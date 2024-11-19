package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	ref "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/search"
	"silvatek.uk/trustedassertions/internal/statements"
)

type FireStore struct {
	projectId    string
	databaseName string
}

const MainCollection = "Primary"
const KeyCollection = "Keys"
const UserCollection = "Users"

var EmptyRefs = []ref.HashUri{}

var client *firestore.Client

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
	if client == nil {
		newClient, err := firestore.NewClientWithDatabase(ctx, fs.projectId, fs.databaseName)
		if err != nil {
			log.ErrorfX(ctx, "Error connecting to database: %v", err)
		} else {
			client = newClient
			log.DebugfX(ctx, "Connected to Firestore database: %v", client)
		}
	}
	return client
}

func (fs *FireStore) store(collection string, id string, data map[string]interface{}) {
	ctx := context.TODO()
	client := fs.client(ctx)
	result, err := client.Collection(collection).Doc(id).Set(ctx, data)

	if err != nil {
		log.Errorf("Error writing value: %v", err)
	} else {
		log.Debugf("Written: %s %v", id, result)
	}
}

func rawDataMap(uri ref.HashUri, content string, summary string, searchText string) map[string]interface{} {
	data := make(map[string]interface{})

	data["uri"] = uri.String()
	data["content"] = content
	data["datatype"] = uri.Kind()
	data["updated"] = time.Now().Format(time.RFC3339)

	if summary != "" {
		data["summary"] = summary
	}
	if searchText != "" {
		data["words"] = search.SearchWords(searchText)
	}

	return data
}

func contentDataMap(value ref.Referenceable) map[string]interface{} {
	uri := value.Uri()
	if value.Type() != "" && !uri.HasType() {
		uri = uri.WithType(value.Type())
	}
	data := rawDataMap(uri, value.Content(), value.Summary(), value.TextContent())
	return data
}

func (fs *FireStore) StoreRecord(ctx context.Context, uri ref.HashUri, rec DbRecord) {
	client := fs.client(ctx)

	result, err := client.Collection(MainCollection).Doc(uri.Escaped()).Set(ctx, rec)

	if err != nil {
		log.ErrorfX(ctx, "Error writing value: %v", err)
	} else {
		log.DebugfX(ctx, "Written: %s %v", uri.Escaped(), result)
	}
}

func (fs *FireStore) StoreRaw(uri ref.HashUri, content string) {
	log.Debugf("Writing to datastore: %s", uri)

	fs.store(MainCollection, uri.Escaped(), rawDataMap(uri, content, "", ""))
}

func (fs *FireStore) Store(ctx context.Context, value ref.Referenceable) {
	log.DebugfX(ctx, "Writing to datastore: %s", value.Uri())

	uri := value.Uri()
	if value.Type() != "" && !uri.HasType() {
		uri = uri.WithType(value.Type())
	}

	rec := DbRecord{
		Uri:         uri.String(),
		Content:     value.Content(),
		DataType:    value.Type(),
		Summary:     value.Summary(),
		Updated:     time.Now().Format(time.RFC3339),
		SearchWords: search.SearchWords(value.TextContent()),
	}
	fs.StoreRecord(ctx, uri, rec)
}

// func (fs *FireStore) storeRefs(ctx context.Context, refs []Reference) {
// 	for _, ref := range refs {
// 		fs.StoreRef(ctx, ref)
// 	}
// }

func (fs *FireStore) StoreKey(entityUri ref.HashUri, key string) {
	data := make(map[string]interface{})
	data["entity"] = entityUri.Unadorned()
	data["encoding"] = "base64"
	data["key"] = key

	fs.store(KeyCollection, entityUri.Escaped(), data)
}

func (fs *FireStore) StoreRef(ctx context.Context, reference ref.Reference) {
	client := fs.client(ctx)

	data := make(map[string]string)
	data["source"] = reference.Source.String()
	data["target"] = reference.Target.String()
	data["summary"] = reference.Summary
	data["updated"] = time.Now().Format(time.RFC3339)

	refs := client.Collection(MainCollection).Doc(reference.Target.Escaped()).Collection("refs")
	refs.Doc(reference.Source.Escaped()).Set(ctx, data)

	log.DebugfX(ctx, "Stored reference from %s to %s", reference.Source.String(), reference.Target.String())
}

func (fs *FireStore) fetch(ctx context.Context, uri ref.HashUri) (*DbRecord, error) {
	client := fs.client(ctx)

	doc, err := client.Collection(MainCollection).Doc(uri.Escaped()).Get(ctx)
	if err != nil {
		log.ErrorfX(ctx, "Error reading value: %v", err)
		return nil, err
	} else {
		record := DbRecord{}
		doc.DataTo(&record)
		return &record, nil
	}
}

func (fs *FireStore) Fetch(ctx context.Context, uri ref.HashUri) (ref.Referenceable, error) {
	record, err := fs.fetch(ctx, uri)
	if err != nil {
		return ref.REF_ERROR, err
	}

	item := assertions.NewReferenceable(record.DataType)
	item.ParseContent(record.Content)

	return item, nil
}

func (fs *FireStore) FetchStatement(ctx context.Context, uri ref.HashUri) (statements.Statement, error) {
	record, err := fs.fetch(ctx, uri)

	log.DebugfX(ctx, "Fetched statement %s", uri)

	if err != nil {
		return *statements.NewStatement("{bad record}"), err
	} else {
		return *statements.NewStatement(record.Content), nil
	}
}

func (fs *FireStore) FetchEntity(ctx context.Context, uri ref.HashUri) (entities.Entity, error) {
	record, err := fs.fetch(ctx, uri)

	log.DebugfX(ctx, "Fetched entity %s", uri)

	if err != nil {
		return entities.NewEntity("{bad record}", *big.NewInt(0)), err
	} else {
		return entities.ParseCertificate(record.Content), nil
	}
}

func (fs *FireStore) FetchAssertion(ctx context.Context, uri ref.HashUri) (assertions.Assertion, error) {
	record, err := fs.fetch(ctx, uri)

	if err != nil {
		return assertions.NewAssertion("{bad record}"), err
	}

	log.DebugfX(ctx, "Fetched assertion %s", uri)

	assertion, err := assertions.ParseAssertionJwt(record.Content)
	if err != nil {
		log.Errorf("Error parsing JWT: %v", err)
		return assertions.NewAssertion("{bad record}"), err
	}
	return assertion, nil
}

func (fs *FireStore) FetchDocument(ctx context.Context, uri ref.HashUri) (docs.Document, error) {
	record, _ := fs.fetch(ctx, uri)

	log.DebugfX(ctx, "Fetched document %s", uri)

	doc, _ := docs.MakeDocument(record.Content)

	return *doc, nil
}

// func (fs *FireStore) FetchMany(ctx context.Context, keys []ref.HashUri) ([]ref.Referenceable, error) {
// 	log.DebugfX(ctx, "Fetching %d keys", len(keys))
// 	results := make([]ref.Referenceable, 0)

// 	refs := make([]*firestore.DocumentRef, len(keys))
// 	for n, key := range keys {
// 		refs[n] = client.Collection(MainCollection).Doc(key.Escaped())
// 	}

// 	records, err := fs.query(ctx, firestore.DocumentID, "in", refs)
// 	if err != nil {
// 		return results, err
// 	}
// 	for _, record := range records {
// 		value := assertions.Newref.Referenceable(record.DataType)
// 		value.ParseContent(record.Content)
// 		results = append(results, value)
// 	}

// 	log.DebugfX(ctx, "Fetched %d values", len(results))

// 	return results, nil
// }

type KeyRecord struct {
	Entity   string `json:"entity"`
	Key      string `json:"key"`
	Encoding string `json:"encoding"`
}

func (fs *FireStore) FetchKey(entityUri ref.HashUri) (string, error) {
	ctx := context.TODO()
	client := fs.client(ctx)

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

type DbReference struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

func (fs *FireStore) FetchRefs(ctx context.Context, uri ref.HashUri) ([]ref.Reference, error) {
	log.DebugfX(ctx, "Fetching references for %s", uri.String())

	client := fs.client(ctx)
	results := make([]ref.Reference, 0)

	refs := client.Collection(MainCollection).Doc(uri.Escaped()).Collection("refs").Documents(ctx)
	for {
		doc, err := refs.Next()
		if err == iterator.Done {
			break
		}
		record := DbReference{}
		doc.DataTo(&record)

		reference := ref.Reference{
			Source:  ref.UriFromString(record.Source),
			Target:  ref.UriFromString(record.Target),
			Summary: record.Summary,
		}

		results = append(results, reference)
	}

	log.DebugfX(ctx, "Found %d references", len(results))
	return results, nil
}

func (fs *FireStore) StoreUser(ctx context.Context, user auth.User) {
	client := fs.client(ctx)

	client.Collection(UserCollection).Doc(user.Id).Set(ctx, user)

	log.Debugf("Stored user %s", user.Id)
}

func (fs *FireStore) FetchUser(ctx context.Context, id string) (auth.User, error) {
	client := fs.client(ctx)

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

	queryWords := search.SearchWords(query)

	results := make([]SearchResult, 0)

	records, _ := fs.query(ctx, "words", "array-contains-any", queryWords)
	for _, record := range records {
		uri := ref.UriFromString(record.Uri)
		if !uri.HasType() {
			uri = uri.WithType(record.DataType)
		}

		result := SearchResult{
			Uri:       uri,
			Content:   record.Summary,
			Relevance: 0.8,
		}
		results = append(results, result)
	}

	return results, nil
}

func (fs *FireStore) query(ctx context.Context, fieldName string, operator string, values interface{}) ([]DbRecord, error) {
	client := fs.client(ctx)

	results := make([]DbRecord, 0)

	query := client.Collection(MainCollection).Where(fieldName, operator, values).WithRunOptions(firestore.ExplainOptions{Analyze: true})
	docs := query.Documents(ctx)
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return results, err
		}

		record := DbRecord{}
		doc.DataTo(&record)
		results = append(results, record)
	}
	plan, err := docs.ExplainMetrics()
	if err != nil {
		log.ErrorfX(ctx, "Error in query explain: %v", err)
	} else {
		log.DebugfX(ctx, "Plan summary: %s", explainString(plan))
	}

	return results, nil
}

func explainString(plan *firestore.ExplainMetrics) string {
	var sb strings.Builder
	sb.WriteString("Execution duration=")
	sb.WriteString(plan.ExecutionStats.ExecutionDuration.String())
	sb.WriteString("\nRead operations=")
	sb.WriteString(strconv.Itoa(int(plan.ExecutionStats.ReadOperations)))
	sb.WriteString("\nIndexes used=")
	sb.WriteString(strconv.Itoa(len(plan.PlanSummary.IndexesUsed)))
	sb.WriteString("\nJSON=")
	bytes, _ := json.Marshal(plan)
	sb.Write(bytes)

	return sb.String()
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
			summary = entities.ParseCertificate(record.Content).CommonName
		}
		log.Debugf("Searching %s => %s", record.Uri, summary)

		if contentMatches(summary, query) {
			uri := ref.UriFromString(record.Uri)
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

type SearchData struct {
	Uri   string   `json:"uri"`
	Words []string `json:"words"`
}

func (fs *FireStore) Reindex() {
	log.Info("Reindexing...")
	ctx := context.TODO()
	client := fs.client(ctx)
	// defer client.Close()

	docs := client.Collection(MainCollection).Documents(ctx)
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}

		record := DbRecord{}
		doc.DataTo(&record)

		if record.DataType == "assertion" {
			continue
		}

		summary := record.Summary
		words := search.SearchWords(summary)

		doc.Ref.Update(ctx, []firestore.Update{
			{
				Path:  "words",
				Value: words,
			},
		})

	}

	log.Info("Reindex complete.")
}

func (fs *FireStore) StoreRegistration(ctx context.Context, reg auth.Registration) error {
	return nil
}

func (fs *FireStore) FetchRegistration(ctx context.Context, code string) (auth.Registration, error) {
	return auth.Registration{}, errors.New("Registration not found with code " + code)
}

package datastore

import (
	"errors"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type DataStore interface {
	Name() string
	AutoInit() bool
	Store(value assertions.Referenceable)
	StoreRaw(uri assertions.HashUri, content string)
	StoreKey(entityUri assertions.HashUri, key string)
	StoreRef(source assertions.HashUri, target assertions.HashUri, refType string)
	StoreUser(user auth.User)
	FetchStatement(key assertions.HashUri) (assertions.Statement, error)
	FetchEntity(key assertions.HashUri) (assertions.Entity, error)
	FetchAssertion(key assertions.HashUri) (assertions.Assertion, error)
	FetchDocument(key assertions.HashUri) (assertions.Document, error)
	FetchKey(entityUri assertions.HashUri) (string, error)
	FetchRefs(key assertions.HashUri) ([]assertions.HashUri, error)
	FetchUser(id string) (auth.User, error)
	Search(query string) ([]SearchResult, error)
	Reindex()
}

type SearchResult struct {
	Uri       assertions.HashUri
	Content   string
	Relevance float32
}

type KeyNotFoundError struct {
}

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

type InMemoryDataStore struct {
	data  map[string]string
	keys  map[string]string
	refs  map[string][]string
	users map[string]auth.User
	krefs map[string]auth.KeyRef
}

var ActiveDataStore DataStore

func InitInMemoryDataStore() {
	datastore := InMemoryDataStore{}
	datastore.data = make(map[string]string)
	datastore.keys = make(map[string]string)
	datastore.refs = make(map[string][]string)
	datastore.users = make(map[string]auth.User)
	datastore.krefs = make(map[string]auth.KeyRef)
	ActiveDataStore = &datastore
}

func (ds *InMemoryDataStore) Name() string {
	return "InMemoryDataStore"
}

func (ds *InMemoryDataStore) AutoInit() bool {
	return true
}

func (ds *InMemoryDataStore) StoreRaw(uri assertions.HashUri, content string) {
	log.Debugf("Storing %s", uri)
	ds.data[uri.Escaped()] = content
}

func (ds *InMemoryDataStore) Store(value assertions.Referenceable) {
	ds.StoreRaw(value.Uri(), value.Content())
}

func (ds *InMemoryDataStore) StoreKey(entityUri assertions.HashUri, key string) {
	ds.keys[entityUri.Escaped()] = key
}

func (ds *InMemoryDataStore) StoreRef(source assertions.HashUri, target assertions.HashUri, refType string) {
	refs, ok := ds.refs[target.Escaped()]
	if !ok {
		refs = make([]string, 0)
	}
	refs = append(refs, source.Escaped())
	ds.refs[target.Escaped()] = refs
	log.Debugf("Stored reference from %s to %s", source.Short(), target.Short())
}

func (ds *InMemoryDataStore) FetchStatement(key assertions.HashUri) (assertions.Statement, error) {
	content, ok := ds.data[key.Escaped()]
	if !ok {
		return assertions.Statement{}, errors.New("statement not found: " + key.String())
	}
	return assertions.NewStatement(content), nil
}

func (ds *InMemoryDataStore) FetchEntity(key assertions.HashUri) (assertions.Entity, error) {
	content := ds.data[key.Escaped()]
	return assertions.ParseCertificate(content), nil
}

func (ds *InMemoryDataStore) FetchAssertion(key assertions.HashUri) (assertions.Assertion, error) {
	content := ds.data[key.Escaped()]
	assertion, err := assertions.ParseAssertionJwt(content)
	if err != nil {
		log.Errorf("Error parsing assertion JWT: %v", err)
	}
	return assertion, err
}

func (ds *InMemoryDataStore) FetchDocument(key assertions.HashUri) (assertions.Document, error) {
	content := ds.data[key.Escaped()]
	doc, _ := assertions.MakeDocument(content)
	return *doc, nil
}

func (ds *InMemoryDataStore) FetchKey(entityUri assertions.HashUri) (string, error) {
	key, ok := ds.keys[entityUri.Escaped()]
	if !ok {
		return "", errors.New("entity id not found " + entityUri.String())
	}
	return key, nil
}

func (ds *InMemoryDataStore) FetchRefs(key assertions.HashUri) ([]assertions.HashUri, error) {
	uris := make([]assertions.HashUri, 0)
	result, ok := ds.refs[key.Escaped()]
	if !ok {
		return uris, nil
	}
	for _, u := range result {
		uri := assertions.UnescapeUri(u, "assertion")
		//uri := assertions.UriFromString(u).WithType("assertion")
		uris = append(uris, uri)
	}
	return uris, nil
}

func (ds *InMemoryDataStore) StoreUser(user auth.User) {
	ds.users[user.Id] = user
	if user.KeyRefs != nil {
		for _, ref := range user.KeyRefs {
			ds.krefs[ref.UserId+" "+ref.KeyId] = ref
		}
	}
}

func (ds *InMemoryDataStore) FetchUser(id string) (auth.User, error) {
	user, ok := ds.users[id]
	if !ok {
		return auth.User{}, errors.New("User not found with id " + id)
	}
	user.KeyRefs = make([]auth.KeyRef, 0)
	for key, value := range ds.krefs {
		if strings.HasPrefix(key, id) {
			user.KeyRefs = append(user.KeyRefs, value)
		}
	}
	return user, nil
}

func (ds *InMemoryDataStore) Search(query string) ([]SearchResult, error) {
	results := make([]SearchResult, 0)
	query = strings.ToLower(query)
	for key, value := range ds.data {
		if strings.Contains(strings.ToLower(value), query) {
			uri := assertions.UnescapeUri(key, assertions.GuessContentType(value))
			result := SearchResult{
				Uri:       uri,
				Content:   summarise(uri, value),
				Relevance: 0.8,
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func summarise(uri assertions.HashUri, content string) string {
	kind := strings.ToLower(uri.Kind())
	switch kind {
	case "statement":
		return leftChars(content, 100)
	case "entity":
		entity := assertions.ParseCertificate(content)
		return entity.CommonName
	case "document":
		doc, _ := assertions.MakeDocument(content)
		return doc.Summary()
	default:
		return content
	}
}

func leftChars(text string, maxChars int) string {
	if len(text) > maxChars {
		return text[0 : maxChars-1]
	} else {
		return text
	}
}

func (ds *InMemoryDataStore) Reindex() {
	// NO-OP
}

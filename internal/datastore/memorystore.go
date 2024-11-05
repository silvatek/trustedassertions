package datastore

import (
	"errors"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type InMemoryDataStore struct {
	data  map[string]DbRecord
	keys  map[string]string
	refs  map[string][]string
	users map[string]auth.User
	krefs map[string]auth.KeyRef
}

func InitInMemoryDataStore() {
	datastore := InMemoryDataStore{}
	datastore.data = make(map[string]DbRecord)
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

func (ds *InMemoryDataStore) StoreRecord(uri assertions.HashUri, rec DbRecord) {
	log.Debugf("Storing %s", uri)
	ds.data[uri.Escaped()] = rec
}

func (ds *InMemoryDataStore) StoreRaw(uri assertions.HashUri, content string) {
	ds.StoreRecord(uri, DbRecord{Uri: uri.String(), Content: content})
}

func (ds *InMemoryDataStore) Store(value assertions.Referenceable) {
	ds.StoreRecord(value.Uri(), DbRecord{Uri: value.Uri().String(), Content: value.Content()})
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

func (ds *InMemoryDataStore) FetchInto(key assertions.HashUri, item assertions.Referenceable) error {
	record, ok := ds.data[key.Escaped()]
	if !ok {
		return errors.New("URI not found: " + key.String())
	}
	return item.ParseContent(record.Content)
}

func (ds *InMemoryDataStore) FetchStatement(key assertions.HashUri) (assertions.Statement, error) {
	var statement assertions.Statement
	return statement, ds.FetchInto(key, &statement)
}

func (ds *InMemoryDataStore) FetchEntity(key assertions.HashUri) (assertions.Entity, error) {
	var entity assertions.Entity
	return entity, ds.FetchInto(key, &entity)
}

func (ds *InMemoryDataStore) FetchAssertion(key assertions.HashUri) (assertions.Assertion, error) {
	var assertion assertions.Assertion
	return assertion, ds.FetchInto(key, &assertion)
}

func (ds *InMemoryDataStore) FetchDocument(key assertions.HashUri) (assertions.Document, error) {
	var doc assertions.Document
	return doc, ds.FetchInto(key, &doc)
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
		if strings.Contains(strings.ToLower(value.Content), query) {
			uri := assertions.UnescapeUri(key, assertions.GuessContentType(value.Content))
			result := SearchResult{
				Uri:       uri,
				Content:   summarise(uri, value.Content),
				Relevance: 0.8,
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func (ds *InMemoryDataStore) Reindex() {
	// NO-OP
}

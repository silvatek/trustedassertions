package datastore

import (
	"context"
	"errors"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	. "silvatek.uk/trustedassertions/internal/references"
	refs "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

type InMemoryDataStore struct {
	data  map[string]DbRecord
	keys  map[string]string
	refs  map[string][]Reference
	users map[string]auth.User
	krefs map[string]auth.KeyRef
}

func InitInMemoryDataStore() {
	datastore := InMemoryDataStore{}
	datastore.data = make(map[string]DbRecord)
	datastore.keys = make(map[string]string)
	datastore.refs = make(map[string][]Reference)
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

func (ds *InMemoryDataStore) StoreRecord(uri HashUri, rec DbRecord) {
	log.Debugf("Storing %s", uri)
	ds.data[uri.Escaped()] = rec
}

func (ds *InMemoryDataStore) StoreRaw(uri HashUri, content string) {
	ds.StoreRecord(uri, DbRecord{Uri: uri.String(), Content: content})
}

func (ds *InMemoryDataStore) Store(ctx context.Context, value Referenceable) {
	ds.StoreRecord(value.Uri(), DbRecord{Uri: value.Uri().String(), DataType: value.Type(), Content: value.Content()})
}

func (ds *InMemoryDataStore) StoreKey(entityUri HashUri, key string) {
	ds.keys[entityUri.Escaped()] = key
}

func (ds *InMemoryDataStore) StoreRef(ctx context.Context, reference refs.Reference) {
	targetKey := reference.Target.Escaped()
	refs, ok := ds.refs[targetKey]
	if !ok {
		refs = make([]Reference, 0)
	}
	refs = append(refs, reference)
	ds.refs[targetKey] = refs
}

func (ds *InMemoryDataStore) FetchInto(key HashUri, item Referenceable) error {
	record, ok := ds.data[key.Escaped()]
	if !ok {
		return errors.New("URI not found: " + key.String())
	}
	return item.ParseContent(record.Content)
}

func (ds *InMemoryDataStore) FetchStatement(ctx context.Context, key HashUri) (statements.Statement, error) {
	var statement statements.Statement
	return statement, ds.FetchInto(key, &statement)
}

func (ds *InMemoryDataStore) FetchEntity(ctx context.Context, key HashUri) (entities.Entity, error) {
	var entity entities.Entity
	return entity, ds.FetchInto(key, &entity)
}

func (ds *InMemoryDataStore) FetchAssertion(ctx context.Context, key HashUri) (assertions.Assertion, error) {
	var assertion assertions.Assertion
	return assertion, ds.FetchInto(key, &assertion)
}

func (ds *InMemoryDataStore) FetchDocument(ctx context.Context, key HashUri) (docs.Document, error) {
	var doc docs.Document
	return doc, ds.FetchInto(key, &doc)
}

func (ds *InMemoryDataStore) FetchKey(entityUri HashUri) (string, error) {
	key, ok := ds.keys[entityUri.Escaped()]
	if !ok {
		return "", errors.New("entity id not found " + entityUri.String())
	}
	return key, nil
}

func (ds *InMemoryDataStore) FetchRefs(ctx context.Context, key HashUri) ([]Reference, error) {
	refs := make([]Reference, 0)
	result, ok := ds.refs[key.Escaped()]
	if !ok {
		return refs, nil
	}
	// for _, ref := range result {
	// 	refs = append(refs, ref)
	// }
	refs = append(refs, result...)
	return refs, nil
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
			uri := UnescapeUri(key, assertions.GuessContentType(value.Content))
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

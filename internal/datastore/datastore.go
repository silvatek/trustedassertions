package datastore

import (
	"context"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/logging"
	refs "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

type DataStore interface {
	Name() string
	AutoInit() bool

	Fetch(ctx context.Context, uri refs.HashUri) (refs.Referenceable, error)
	FetchStatement(ctx context.Context, key refs.HashUri) (statements.Statement, error)
	FetchEntity(ctx context.Context, key refs.HashUri) (entities.Entity, error)
	FetchAssertion(ctx context.Context, key refs.HashUri) (assertions.Assertion, error)
	FetchDocument(ctx context.Context, key refs.HashUri) (docs.Document, error)
	Store(ctx context.Context, value refs.Referenceable)
	StoreRaw(uri refs.HashUri, content string)

	FetchRefs(ctx context.Context, key refs.HashUri) ([]refs.Reference, error)
	StoreRef(ctx context.Context, reference refs.Reference)

	StoreKey(entityUri refs.HashUri, key string)
	StoreUser(ctx context.Context, user auth.User)
	StoreRegistration(ctx context.Context, reg auth.Registration) error

	FetchKey(entityUri refs.HashUri) (string, error)
	FetchUser(ctx context.Context, id string) (auth.User, error)
	FetchRegistration(ctx context.Context, code string) (auth.Registration, error)

	Search(ctx context.Context, query string) ([]SearchResult, error)

	Reindex()
}

type DbRecord struct {
	Uri         string   `json:"uri" firestore:"uri"`
	Content     string   `json:"content" firestore:"content"`
	DataType    string   `json:"datatype" firestore:"datatype"`
	Summary     string   `json:"summary" firestore:"summary"`
	Updated     string   `json:"updated" firestore:"updated"`
	SearchWords []string `json:"words" firestore:"words"`
}

type SearchResult struct {
	Uri       refs.HashUri
	Content   string
	Relevance float32
}

type KeyNotFoundError struct {
}

var log = logging.GetLogger("datastore")

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

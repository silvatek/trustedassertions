package datastore

import (
	"context"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	refs "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

type DataStore interface {
	Name() string
	AutoInit() bool
	Store(ctx context.Context, value refs.Referenceable)
	StoreRaw(uri refs.HashUri, content string)
	StoreKey(entityUri refs.HashUri, key string)
	StoreRef(source refs.HashUri, target refs.HashUri, refType string)
	StoreUser(user auth.User)
	FetchMany(ctx context.Context, uris []refs.HashUri) ([]refs.Referenceable, error)
	FetchStatement(ctx context.Context, key refs.HashUri) (statements.Statement, error)
	FetchEntity(ctx context.Context, key refs.HashUri) (entities.Entity, error)
	FetchAssertion(ctx context.Context, key refs.HashUri) (assertions.Assertion, error)
	FetchDocument(ctx context.Context, key refs.HashUri) (docs.Document, error)
	FetchKey(entityUri refs.HashUri) (string, error)
	FetchRefs(ctx context.Context, key refs.HashUri) ([]refs.Reference, error)
	FetchUser(id string) (auth.User, error)
	Search(query string) ([]SearchResult, error)
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

func (e *KeyNotFoundError) Error() string {
	return "Key not found"
}

func summarise(uri refs.HashUri, content string) string {
	kind := strings.ToLower(uri.Kind())
	switch kind {
	case "statement":
		return leftChars(content, 100)
	case "entity":
		entity := entities.ParseCertificate(content)
		return entity.CommonName
	case "document":
		doc, _ := docs.MakeDocument(content)
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

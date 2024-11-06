package datastore

import (
	"context"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
)

type DataStore interface {
	Name() string
	AutoInit() bool
	Store(ctx context.Context, value assertions.Referenceable)
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

type DbRecord struct {
	Uri         string   `json:"uri" firestore:"uri"`
	Content     string   `json:"content" firestore:"content"`
	DataType    string   `json:"datatype" firestore:"datatype"`
	Summary     string   `json:"summary" firestore:"summary"`
	Updated     string   `json:"updated" firestore:"updated"`
	SearchWords []string `json:"words" firestore:"words"`
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

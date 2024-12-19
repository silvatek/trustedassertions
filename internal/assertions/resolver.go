package assertions

import (
	"context"
	"errors"
	"strings"

	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

// Resolver is responsible for fetching the data associated with a Hash URI.
type Resolver interface {
	FetchStatement(ctx context.Context, key HashUri) (statements.Statement, error)
	FetchEntity(ctx context.Context, key HashUri) (entities.Entity, error)
	FetchAssertion(ctx context.Context, key HashUri) (Assertion, error)
	FetchDocument(ctx context.Context, key HashUri) (docs.Document, error)
	FetchKey(entityUri HashUri) (string, error)
	FetchRefs(ctx context.Context, key HashUri) ([]Reference, error)
}

type NullResolver struct{}

var ErrNotImplemented = errors.New("not implemented")

func (r NullResolver) FetchStatement(ctx context.Context, key HashUri) (statements.Statement, error) {
	return statements.Statement{}, ErrNotImplemented
}

func (r NullResolver) FetchEntity(ctx context.Context, key HashUri) (entities.Entity, error) {
	return entities.Entity{}, ErrNotImplemented
}

func (r NullResolver) FetchAssertion(ctx context.Context, key HashUri) (Assertion, error) {
	return Assertion{}, ErrNotImplemented
}

func (r NullResolver) FetchDocument(ctx context.Context, key HashUri) (docs.Document, error) {
	return docs.Document{}, ErrNotImplemented
}

func (r NullResolver) FetchKey(key HashUri) (string, error) {
	return "", ErrNotImplemented
}

func (r NullResolver) FetchRefs(ctx context.Context, key HashUri) ([]Reference, error) {
	return []Reference{}, ErrNotImplemented
}

func NewReferenceable(kind string) Referenceable {
	switch strings.ToLower(kind) {
	case "statement":
		return &statements.Statement{}
	case "entity":
		return &entities.Entity{}
	case "assertion":
		return &Assertion{}
	case "document":
		return &docs.Document{}
	default:
		return nil
	}
}

func GuessContentType(content string) string {
	if strings.HasPrefix(content, "<?xml") && strings.Contains(content, "<document>") {
		return "Document"
	}
	if len(content) < 512 {
		// Both X509 certificates and JWTs signed by Entities are longer than 512 characters
		return "Statement"
	}
	if strings.HasPrefix(content, "-----BEGIN CERTIFICATE----") {
		// X509 certificates for Entities are self-describing
		return "Entity"
	}
	if strings.HasPrefix(content, "eyJ") {
		// Assertion JWTs start with bas64-encoded "{"
		return "Assertion"
	}
	return "Statement"
}

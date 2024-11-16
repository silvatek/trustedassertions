package assertions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

// Resolver is responsible for fetching the data associated with a Hash URI.
type Resolver interface {
	FetchStatement(ctx context.Context, key HashUri) (statements.Statement, error)
	FetchEntity(ctx context.Context, key HashUri) (entities.Entity, error)
	FetchAssertion(ctx context.Context, key HashUri) (Assertion, error)
	FetchDocument(ctx context.Context, key HashUri) (docs.Document, error)
	FetchMany(ctx context.Context, keys []HashUri) ([]Referenceable, error)
	FetchKey(entityUri HashUri) (string, error)
	FetchRefs(ctx context.Context, key HashUri) ([]HashUri, error)
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

func (r NullResolver) FetchRefs(ctx context.Context, key HashUri) ([]HashUri, error) {
	return []HashUri{}, ErrNotImplemented
}

func (r NullResolver) FetchMany(ctx context.Context, keys []HashUri) ([]Referenceable, error) {
	return []Referenceable{}, ErrNotImplemented
}

type ReferenceSummary struct {
	Uri     HashUri
	Summary string
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

func SummariseAssertion(ctx context.Context, assertion Assertion, currentUri HashUri, resolver Resolver) string {
	if assertion.Issuer == "" {
		var err error
		assertion, err = ParseAssertionJwt(assertion.Content())
		if err != nil {
			return "Error parsing JWT"
		}
	}

	subjectUri := UriFromString(assertion.Subject)
	if subjectUri.Equals(currentUri) {
		issuer, _ := resolver.FetchEntity(ctx, UriFromString(assertion.Issuer))
		return fmt.Sprintf("%s asserts this %s", issuer.CommonName, assertion.Category)
	}

	issuerUri := UriFromString(assertion.Issuer)
	if issuerUri.Equals(currentUri) {
		statement, _ := resolver.FetchStatement(ctx, UriFromString(assertion.Subject))
		return fmt.Sprintf("This entity asserts that statement %s %s", statement.Uri().Short(), assertion.Category)
	}

	return "Some kind of assertion"
}

func EnrichReferences(ctx context.Context, uris []HashUri, currentUri HashUri, resolver Resolver) []ReferenceSummary {
	summaries := make([]ReferenceSummary, 0)

	values, err := resolver.FetchMany(ctx, uris)
	if err != nil {
		log.ErrorfX(ctx, "Error fetching many values: %v", err)
	}
	for _, value := range values {
		var summary string
		switch strings.ToLower(value.Type()) {
		case "assertion":
			assertion := value.(*Assertion)
			summary = SummariseAssertion(ctx, *assertion, currentUri, resolver)
		case "document":
			document := value.(*docs.Document)
			summary = document.Metadata.Title
		default:
			summary = "unknown " + value.Type()
		}
		ref := ReferenceSummary{Uri: value.Uri(), Summary: summary}
		summaries = append(summaries, ref)
	}

	// for _, uri := range uris {
	// 	var summary string
	// 	switch uri.Kind() {
	// 	case "assertion":
	// 		assertion, err := resolver.FetchAssertion(ctx, uri)
	// 		if err != nil {
	// 			log.ErrorfX(ctx, "Error fetching assertion %s %v", uri.String(), err)
	// 			summary = "Error fetching assertion"
	// 		} else {
	// 			summary = SummariseAssertion(ctx, assertion, currentUri, resolver)
	// 		}
	// 	case "document":
	// 		document, err := resolver.FetchDocument(ctx, uri)
	// 		if err != nil {
	// 			log.ErrorfX(ctx, "Error fetching document %s %v", uri.String(), err)
	// 			summary = "Error fetching document"
	// 		} else {
	// 			summary = document.Metadata.Title
	// 		}
	// 	default:
	// 		summary = "unknown " + uri.Kind()
	// 	}
	// 	ref := ReferenceSummary{Uri: uri, Summary: summary}
	// 	summaries = append(summaries, ref)
	// }

	return summaries
}

func GuessContentType(content string) string {
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
	if strings.HasPrefix(content, "<?xml") && strings.Contains(content, "<document>") {
		return "Document"
	}
	return "Statement"
}

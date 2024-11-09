package assertions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	log "silvatek.uk/trustedassertions/internal/logging"
)

// Referenceable is a core data type that can be referenced by an assertion.
type Referenceable interface {
	Uri() HashUri
	Type() string
	Content() string
	Summary() string
	TextContent() string
	References() []HashUri
	ParseContent(content string) error
}

// Resolver is responsible for fetching the data associated with a Hash URI.
type Resolver interface {
	FetchStatement(ctx context.Context, key HashUri) (Statement, error)
	FetchEntity(ctx context.Context, key HashUri) (Entity, error)
	FetchAssertion(ctx context.Context, key HashUri) (Assertion, error)
	FetchDocument(ctx context.Context, key HashUri) (Document, error)
	FetchMany(ctx context.Context, keys []HashUri) ([]Referenceable, error)
	FetchKey(entityUri HashUri) (string, error)
	FetchRefs(ctx context.Context, key HashUri) ([]HashUri, error)
}

type NullResolver struct{}

var ErrNotImplemented = errors.New("not implemented")

func (r NullResolver) FetchStatement(ctx context.Context, key HashUri) (Statement, error) {
	return Statement{}, ErrNotImplemented
}

func (r NullResolver) FetchEntity(ctx context.Context, key HashUri) (Entity, error) {
	return Entity{}, ErrNotImplemented
}

func (r NullResolver) FetchAssertion(ctx context.Context, key HashUri) (Assertion, error) {
	return Assertion{}, ErrNotImplemented
}

func (r NullResolver) FetchDocument(ctx context.Context, key HashUri) (Document, error) {
	return Document{}, ErrNotImplemented
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
		return &Statement{}
	case "entity":
		return &Entity{}
	case "assertion":
		return &Assertion{}
	case "document":
		return &Document{}
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
			document := value.(*Document)
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

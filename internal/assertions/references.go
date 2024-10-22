package assertions

import (
	"errors"
	"fmt"
)

// Referenceable is a core data type that can be referenced by an assertion.
type Referenceable interface {
	Uri() HashUri
	Type() string
	Content() string
	Summary() string
}

// Resolver is responsible for fetching the data associated with a Hash URI.
type Resolver interface {
	FetchStatement(key HashUri) (Statement, error)
	FetchEntity(key HashUri) (Entity, error)
	FetchAssertion(key HashUri) (Assertion, error)
	FetchKey(entityUri HashUri) (string, error)
	FetchRefs(key HashUri) ([]HashUri, error)
}

type NullResolver struct{}

var ErrNotImplemented = errors.New("not implemented")

func (r NullResolver) FetchStatement(key HashUri) (Statement, error) {
	return Statement{}, ErrNotImplemented
}

func (r NullResolver) FetchEntity(key HashUri) (Entity, error) {
	return Entity{}, ErrNotImplemented
}

func (r NullResolver) FetchAssertion(key HashUri) (Assertion, error) {
	return Assertion{}, ErrNotImplemented
}

func (r NullResolver) FetchKey(key HashUri) (string, error) {
	return "", ErrNotImplemented
}

func (r NullResolver) FetchRefs(key HashUri) ([]HashUri, error) {
	return []HashUri{}, ErrNotImplemented
}

type ReferenceSummary struct {
	Uri     HashUri
	Summary string
}

func SummariseAssertion(assertion Assertion, currentUri HashUri, resolver Resolver) string {
	if assertion.Issuer == "" {
		var err error
		assertion, err = ParseAssertionJwt(assertion.Content())
		if err != nil {
			return "Error parsing JWT"
		}
	}

	subjectUri := UriFromString(assertion.Subject)
	if subjectUri.Equals(currentUri) {
		issuer, _ := resolver.FetchEntity(UriFromString(assertion.Issuer))
		return fmt.Sprintf("%s asserts this %s", issuer.CommonName, assertion.Category)
	}

	issuerUri := UriFromString(assertion.Issuer)
	if issuerUri.Equals(currentUri) {
		statement, _ := resolver.FetchStatement(UriFromString(assertion.Subject))
		return fmt.Sprintf("This entity asserts that statement %s %s", statement.Uri().Short(), assertion.Category)
	}

	return "Some kind of assertion"
}

func EnrichReferences(uris []HashUri, currentUri HashUri, resolver Resolver) []ReferenceSummary {
	summaries := make([]ReferenceSummary, 0)

	for _, uri := range uris {
		var summary string
		switch uri.Kind() {
		case "assertion":
			assertion, _ := resolver.FetchAssertion(uri)

			summary = SummariseAssertion(assertion, currentUri, resolver)
		default:
			summary = "unknown " + uri.Kind()
		}
		ref := ReferenceSummary{Uri: uri, Summary: summary}
		summaries = append(summaries, ref)
	}

	return summaries
}

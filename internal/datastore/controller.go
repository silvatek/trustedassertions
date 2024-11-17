package datastore

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var ActiveDataStore DataStore

func CreateAssertion(ctx context.Context, statementUri references.HashUri, confidence float64, entity entities.Entity, privateKey *rsa.PrivateKey, kind string) *assertions.Assertion {
	assertion := assertions.NewAssertion(kind)
	assertion.Subject = statementUri.String()
	assertion.IssuedAt = jwt.NewNumericDate(time.Now())
	assertion.NotBefore = assertion.IssuedAt
	assertion.Confidence = float32(confidence)
	assertion.SetAssertingEntity(entity)
	assertion.MakeJwt(privateKey)
	ActiveDataStore.Store(ctx, &assertion)

	StoreReferenceWithSummary(ctx, assertion.Uri(), references.UriFromString(assertion.Subject))
	StoreReferenceWithSummary(ctx, assertion.Uri(), references.UriFromString(assertion.Issuer))

	return &assertion
}

// Adds a summary to a reference and stores it in the active datastore.
func StoreReferenceWithSummary(ctx context.Context, source references.HashUri, target references.HashUri) {
	ref := references.Reference{
		Source: source,
		Target: target,
	}
	MakeSummary(ctx, nil, &ref, ActiveDataStore)
	ActiveDataStore.StoreRef(ctx, ref)
}

func CreateStatementAndAssertion(ctx context.Context, content string, entityUri references.HashUri, confidence float64) (*assertions.Assertion, error) {
	log.DebugfX(ctx, "Creating statement and assertion")

	b64key, err := ActiveDataStore.FetchKey(entityUri)
	if err != nil {
		return nil, err
	}
	privateKey := entities.PrivateKeyFromString(b64key)
	entity, err := ActiveDataStore.FetchEntity(ctx, entityUri)
	if err != nil {
		return nil, err
	}

	// Create and save the statement
	statement := statements.NewStatement(content)
	ActiveDataStore.Store(ctx, statement)

	log.DebugfX(ctx, "Statement created")

	// Create and save an assertion by the default entity that the statement is probably true
	assertion := CreateAssertion(ctx, statement.Uri(), confidence, entity, privateKey, "IsTrue")

	log.DebugfX(ctx, "Assertion created")

	return assertion, nil
}

func MakeSummary(ctx context.Context, target *references.Referenceable, ref *references.Reference, resolver assertions.Resolver) {
	switch ref.Source.Kind() {
	case "statement":
		statement, _ := resolver.FetchStatement(ctx, ref.Source)
		ref.Summary = statement.Summary()
	case "entity":
		entity, _ := ActiveDataStore.FetchEntity(ctx, ref.Source)
		ref.Summary = entity.Summary()
	case "document":
		doc, _ := resolver.FetchDocument(ctx, ref.Source)
		ref.Summary = doc.Summary()
	case "assertion":
		assertion, _ := resolver.FetchAssertion(ctx, ref.Source)

		issuerUri := references.UriFromString(assertion.Issuer)
		var issuerName string

		if target != nil && issuerUri.Equals((*target).Uri()) {
			issuerName = (*target).Summary()
		} else {
			entity, _ := ActiveDataStore.FetchEntity(ctx, issuerUri)
			issuerName = entity.Summary()
		}

		subjectUri := references.UriFromString(assertion.Subject)
		var subjectSummary string

		if target != nil && subjectUri.Equals((*target).Uri()) {
			subjectSummary = (*target).Summary()
		} else {
			subject, _ := resolver.FetchStatement(ctx, subjectUri)
			subjectSummary = subject.Summary()
		}

		ref.Summary = fmt.Sprintf("%s claims that '%s' %s", issuerName, subjectSummary, assertions.CategoryDescription(assertion.Category, "en"))
	default:
		ref.Summary = "Unknown " + ref.Source.Kind()
	}
}

package datastore

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var ActiveDataStore DataStore

func CreateAssertion(ctx context.Context, statementUri references.HashUri, entityUri references.HashUri, kind string, confidence float64, privateKey *rsa.PrivateKey) *assertions.Assertion {
	assertion := assertions.NewAssertion(kind)
	assertion.Subject = statementUri.String()
	assertion.IssuedAt = jwt.NewNumericDate(time.Now())
	assertion.NotBefore = assertion.IssuedAt
	assertion.Confidence = float32(confidence)
	assertion.Issuer = entityUri.String()
	assertion.MakeJwt(privateKey)
	ActiveDataStore.Store(ctx, &assertion)

	CreateReferences(ctx, &assertion)

	return &assertion
}

func CreateReferences(ctx context.Context, target references.Referenceable) {
	for _, uri := range target.References() {
		CreateReferenceWithSummary(ctx, uri, target.Uri())
	}
}

// Creates a reference including a summary and stores it in the active datastore.
func CreateReferenceWithSummary(ctx context.Context, source references.HashUri, target references.HashUri) {
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
	assertion := CreateAssertion(ctx, statement.Uri(), entity.Uri(), "IsTrue", confidence, privateKey)

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
		var assertion assertions.Assertion
		if target != nil && (*target).Uri().Equals(ref.Source) {
			assertion = *((*target).(*assertions.Assertion))
		} else {
			assertion, _ = resolver.FetchAssertion(ctx, ref.Source)
		}
		summary := assertions.SummariseAssertion(ctx, assertion, target, resolver)
		ref.Summary = summary
	default:
		ref.Summary = "Unknown " + ref.Source.Kind()
	}
}

// Creates a new Statement and stores it in the active datastore.
func CreateStatement(ctx context.Context, content string) references.HashUri {
	statement := statements.NewStatement(content)
	ActiveDataStore.Store(ctx, statement)
	return statement.Uri()
}

// Creates a new Entity with a private key and stores both in the active
func CreateEntityWithKey(ctx context.Context, commonName string) references.HashUri {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := entities.Entity{CommonName: commonName}
	entity.MakeCertificate(privateKey)

	ActiveDataStore.Store(ctx, &entity)

	ActiveDataStore.StoreKey(entity.Uri(), entities.PrivateKeyToString(privateKey))

	return entity.Uri()
}

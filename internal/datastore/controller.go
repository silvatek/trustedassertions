package datastore

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/assertions"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var ActiveDataStore DataStore

func CreateAssertion(ctx context.Context, statementUri assertions.HashUri, confidence float64, entity assertions.Entity, privateKey *rsa.PrivateKey, kind string) *assertions.Assertion {
	assertion := assertions.NewAssertion(kind)
	assertion.Subject = statementUri.String()
	assertion.IssuedAt = jwt.NewNumericDate(time.Now())
	assertion.NotBefore = assertion.IssuedAt
	assertion.Confidence = float32(confidence)
	assertion.SetAssertingEntity(entity)
	assertion.MakeJwt(privateKey)
	ActiveDataStore.Store(ctx, &assertion)

	ActiveDataStore.StoreRef(assertion.Uri(), assertions.UriFromString(assertion.Subject), "Assertion.Subject:Statement")
	ActiveDataStore.StoreRef(assertion.Uri(), assertions.UriFromString(assertion.Issuer), "Assertion.Issuer:Entity")

	return &assertion
}

func CreateStatementAndAssertion(ctx context.Context, content string, entityUri assertions.HashUri, confidence float64) (*assertions.Assertion, error) {
	log.DebugfX(ctx, "Creating statement and assertion")

	b64key, err := ActiveDataStore.FetchKey(entityUri)
	if err != nil {
		return nil, err
	}
	privateKey := assertions.StringToPrivateKey(b64key)
	entity, err := ActiveDataStore.FetchEntity(entityUri)
	if err != nil {
		return nil, err
	}

	// Create and save the statement
	statement := assertions.NewStatement(content)
	ActiveDataStore.Store(ctx, statement)

	log.DebugfX(ctx, "Statement created")

	// Create and save an assertion by the default entity that the statement is probably true
	assertion := CreateAssertion(ctx, statement.Uri(), confidence, entity, privateKey, "IsTrue")

	log.DebugfX(ctx, "Assertion created")

	return assertion, nil
}

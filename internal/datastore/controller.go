package datastore

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var ActiveDataStore DataStore

func Summarise(uri references.HashUri, content string) string {
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

func CreateAssertion(ctx context.Context, statementUri references.HashUri, entityUri references.HashUri, kind string, confidence float64, privateKey *rsa.PrivateKey) *assertions.Assertion {
	assertion := assertions.NewAssertion(kind)
	assertion.Subject = statementUri.String()
	assertion.IssuedAt = jwt.NewNumericDate(time.Now())
	assertion.NotBefore = assertion.IssuedAt
	assertion.Confidence = float32(confidence)
	assertion.Issuer = entityUri.String()
	assertion.SetSummary(assertions.SummariseAssertion(ctx, assertion, nil, ActiveDataStore))
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

func CreateStatementAndAssertion(ctx context.Context, content string, entityUri references.HashUri, kind string, confidence float64) (*assertions.Assertion, error) {
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
		cache := make(references.ReferenceMap)
		if target != nil {
			cache[(*target).Uri()] = *target
		}
		summary := assertions.SummariseAssertion(ctx, assertion, cache, resolver)
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

func CreateDocumentAndAssertions(ctx context.Context, content string, entityUri references.HashUri) (*docs.Document, error) {
	entity, err := ActiveDataStore.FetchEntity(ctx, entityUri)
	if err != nil {
		return nil, err
	}

	doc, _ := docs.MakeDocument(content)

	author := &doc.Metadata.Author
	if author.Entity == "" {
		log.DebugfX(ctx, "Setting document author to %s", entity.CommonName)
		author.Entity = entity.Uri().String()
		author.Name = entity.CommonName
	}

	for i := range doc.Sections {
		for j := range doc.Sections[i].Paragraphs {
			for k := range doc.Sections[i].Paragraphs[j].Spans {
				span := &doc.Sections[i].Paragraphs[j].Spans[k]
				if span.Assertion != "" && !strings.HasPrefix(span.Assertion, "hash://") {
					parts := strings.Split(span.Assertion, " ")
					assertionType := parts[0]
					confidence, _ := strconv.ParseFloat(parts[1], 32)

					assertion, _ := CreateStatementAndAssertion(ctx, span.Body, entityUri, assertionType, confidence)

					span.Assertion = assertion.Uri().String()
				}
			}
		}
	}

	doc.UpdateContent()

	ActiveDataStore.Store(ctx, doc)

	for _, uri := range doc.References() {
		ref := references.Reference{
			Source:  doc.Uri(),
			Target:  uri,
			Summary: doc.Summary(),
		}
		ActiveDataStore.StoreRef(ctx, ref)
	}

	return doc, nil
}

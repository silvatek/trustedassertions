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
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var ActiveDataStore DataStore

func CreateAssertion(ctx context.Context, statementUri references.HashUri, entityUri references.HashUri, kind assertions.AssertionType, confidence float64, privateKey *rsa.PrivateKey) *assertions.Assertion {
	statement, _ := ActiveDataStore.FetchStatement(ctx, statementUri)
	entity, _ := ActiveDataStore.FetchEntity(ctx, entityUri)
	return createAssertion(ctx, &statement, &entity, kind, confidence, privateKey)
}

func createAssertion(ctx context.Context, statement *statements.Statement, entity *entities.Entity, kind assertions.AssertionType, confidence float64, privateKey *rsa.PrivateKey) *assertions.Assertion {
	assertion := assertions.NewAssertion(kind)
	assertion.Subject = statement.Uri().String()
	assertion.IssuedAt = jwt.NewNumericDate(time.Now())
	assertion.NotBefore = assertion.IssuedAt
	assertion.Confidence = float32(confidence)
	assertion.Issuer = entity.Uri().String()
	cache := references.ReferenceMap{
		entity.Uri():    entity,
		statement.Uri(): statement,
	}
	assertion.SetSummary(assertions.SummariseAssertion(ctx, assertion, cache, ActiveDataStore))
	assertion.MakeJwt(privateKey)
	ActiveDataStore.Store(ctx, &assertion)

	CreateReferences(ctx, &assertion)

	return &assertion
}

func CreateReferences(ctx context.Context, source references.Referenceable) {
	known := source
	for _, uri := range source.References() {
		ref := references.Reference{
			Source: source.Uri(),
			Target: uri,
		}
		MakeReferenceSummary(ctx, &known, &ref, ActiveDataStore)
		ActiveDataStore.StoreRef(ctx, ref)
	}
}

// Creates a reference including a summary and stores it in the active datastore.
func CreateReferenceWithSummary(ctx context.Context, source references.HashUri, target references.HashUri) {
	ref := references.Reference{
		Source: source,
		Target: target,
	}
	MakeReferenceSummary(ctx, nil, &ref, ActiveDataStore)
	ActiveDataStore.StoreRef(ctx, ref)
}

func CreateStatementAndAssertion(ctx context.Context, content string, entityUri references.HashUri, kind assertions.AssertionType, confidence float64) (*assertions.Assertion, error) {
	b64key, err := ActiveDataStore.FetchKey(entityUri)
	if err != nil {
		return nil, err
	}
	privateKey := entities.PrivateKeyFromString(b64key)
	entity, err := ActiveDataStore.FetchEntity(ctx, entityUri)
	if err != nil {
		return nil, err
	}

	return createStatementAndAssertion(ctx, content, &entity, privateKey, kind, confidence)
}

func createStatementAndAssertion(ctx context.Context, content string, entity *entities.Entity, privateKey *rsa.PrivateKey, kind assertions.AssertionType, confidence float64) (*assertions.Assertion, error) {
	log.DebugfX(ctx, "Creating statement and assertion for %s", entity.Uri())

	statement := statements.NewStatement(content)
	ActiveDataStore.Store(ctx, statement)

	log.DebugfX(ctx, "Statement created %s", statement.Uri())

	assertion := createAssertion(ctx, statement, entity, kind, confidence, privateKey)

	log.DebugfX(ctx, "Assertion created %s", assertion.Uri())

	return assertion, nil
}

// Populates the summary field of a Reference based on the source of the reference.
func MakeReferenceSummary(ctx context.Context, known *references.Referenceable, ref *references.Reference, resolver assertions.Resolver) {
	if known != nil && (*known).Uri().Equals(ref.Source) {
		ref.Summary = (*known).Summary()
		return
	}

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
		cache := make(references.ReferenceMap)
		if known != nil {
			cache[(*known).Uri()] = *known
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
	b64key, err := ActiveDataStore.FetchKey(entityUri)
	if err != nil {
		return nil, err
	}
	privateKey := entities.PrivateKeyFromString(b64key)

	doc, err := docs.MakeDocument(content)
	if err != nil {
		return nil, err
	}

	title := doc.Metadata.Title
	log.InfofX(ctx, "Creating document %q signed by %s", title, entityUri)

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
					assertionType := assertions.AssertionTypeOf(parts[0])
					confidence, _ := strconv.ParseFloat(parts[1], 32)

					assertion, err := createStatementAndAssertion(ctx, span.Body, &entity, privateKey, assertionType, confidence)
					if err != nil {
						return nil, err
					}

					span.Assertion = assertion.Uri().String()
				}
			}
		}
	}

	doc.UpdateContent()

	ActiveDataStore.Store(ctx, doc)

	log.InfofX(ctx, "Creating references for document %q", title)
	CreateReferences(ctx, doc)

	log.InfofX(ctx, "Created document %q as %s", title, doc.Uri())

	return doc, nil
}

func AddPasskey(ctx context.Context, userID string, pk auth.Passkey) error {
	user, err := ActiveDataStore.FetchUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.AddPasskey(pk); err != nil {
		return err
	}
	ActiveDataStore.StoreUser(ctx, user)
	return nil
}

func RemovePasskey(ctx context.Context, userID string, credentialID []byte) error {
	user, err := ActiveDataStore.FetchUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.RemovePasskey(credentialID); err != nil {
		return err
	}
	ActiveDataStore.StoreUser(ctx, user)
	return nil
}

func RecordPasskeyUse(ctx context.Context, userID string, pk auth.Passkey, usedAt time.Time) error {
	user, err := ActiveDataStore.FetchUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.RecordPasskeyUse(pk, usedAt); err != nil {
		return err
	}
	ActiveDataStore.StoreUser(ctx, user)
	return nil
}

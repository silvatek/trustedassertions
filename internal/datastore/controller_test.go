package datastore

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"strings"
	"testing"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/references"
)

func TestMakeSummary(t *testing.T) {
	ctx := context.Background()
	InitInMemoryDataStore()
	assertions.PublicKeyResolver = ActiveDataStore

	issuerUri := CreateEntityWithKey(ctx, "Unit Tester")

	assertion, err := CreateStatementAndAssertion(ctx, "Test 123", issuerUri, "IsTrue", 0.9)
	if err != nil {
		t.Error(err)
	}

	ref := references.Reference{
		Source:  assertion.Uri(),
		Target:  issuerUri,
		Summary: "",
	}

	target := references.Referenceable(assertion)
	MakeReferenceSummary(context.TODO(), &target, &ref, ActiveDataStore)

	if ref.Summary != "Unit Tester claims that 'Test 123' is true" {
		t.Errorf("Unexpected assertion summary: %s", ref.Summary)
	}
}

func TestCreateDocumentAndAssertions(t *testing.T) {
	docContent := `
	<?xml version="1.0" encoding="UTF-8"?>
	<document>
		<metadata>
			<title>Newton Test Doc</title>
		</metadata>
		<section>
			<title>Test 1</title>
			<paragraph>
				<span assertion="IsTrue 0.9">Isaac Newton was a scientist</span>
			</paragraph>		
			<paragraph>
				<span assertion="IsFalse 0.8">Isaac Newton was French</span>
				<span>Some text</span>
			</paragraph>		
		</section>
	</document>
`

	ctx := context.Background()
	InitInMemoryDataStore()
	assertions.PublicKeyResolver = ActiveDataStore

	entityUri := CreateEntityWithKey(ctx, "Unit Tester")

	doc, err := CreateDocumentAndAssertions(ctx, docContent, entityUri)

	if err != nil {
		t.Errorf("Error creating document: %v", err)
		return
	}

	xml := doc.ToXml()
	if !strings.Contains(xml, "<span assertion=\"hash://sha256/") {
		t.Errorf("Did not find assertion in document: %s", xml)
	}

	_, err = ActiveDataStore.FetchDocument(ctx, doc.Uri())
	if err != nil {
		t.Errorf("Could not load document: %v", err)
	}
	_, err = ActiveDataStore.FetchEntity(ctx, references.UriFromString(doc.Metadata.Author.Entity))
	if err != nil {
		t.Errorf("Could not load author entity: %v", err)
	}
	_, err = ActiveDataStore.FetchAssertion(ctx, references.UriFromString(doc.Sections[0].Paragraphs[0].Spans[0].Assertion))
	if err != nil {
		t.Errorf("Could not load assertion 1: %v", err)
	}
	_, err = ActiveDataStore.FetchAssertion(ctx, references.UriFromString(doc.Sections[0].Paragraphs[1].Spans[0].Assertion))
	if err != nil {
		t.Errorf("Could not load assertion 2: %v", err)
	}
}

func TestCreateStatement(t *testing.T) {
	ActiveDataStore = NewInMemoryDataStore()
	ctx := context.Background()

	content := "Testing " + time.Now().Format(time.RFC3339)

	uri := CreateStatement(ctx, content)
	statement, err := ActiveDataStore.FetchStatement(ctx, uri)

	if err != nil {
		t.Errorf("Error creating statement: %v", err)
	}
	if statement.Content() != content {
		t.Errorf("Unexpected statement content: %s", statement.Content())
	}
}

func TestCreateDocumentWithMissingEntity(t *testing.T) {
	ActiveDataStore = NewInMemoryDataStore()
	ctx := context.Background()

	_, err := CreateDocumentAndAssertions(ctx, "<docco/>", references.MakeUri("1234", "entity"))

	if err == nil {
		t.Error("Creating document for missing entity did not fail")
	}
}

func TestCreateAssertionMissingEntity(t *testing.T) {
	ActiveDataStore = NewInMemoryDataStore()
	ctx := context.Background()

	entity := entities.NewEntity("tester", *big.NewInt(1234))
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity.MakeCertificate(privateKey)

	_, err := CreateStatementAndAssertion(ctx, "testing", references.MakeUri("1234", "entity"), "IsTrue", 0.9)

	if err == nil {
		t.Errorf("Unexpected lack of error with no key stored")
	}

	ActiveDataStore.StoreKey(entity.Uri(), entities.PrivateKeyToString(privateKey))

	_, err = CreateStatementAndAssertion(ctx, "testing", entity.Uri(), "IsTrue", 0.9)

	if err == nil {
		t.Errorf("Unexpected lack of error with no entity stored")
	}
}

func TestMakeDocumentReferenceSummary(t *testing.T) {
	ActiveDataStore = NewInMemoryDataStore()
	ctx := context.Background()

	doc, _ := docs.MakeDocument("<document><metadata><title>Test 123</title></metadata></document>")
	ActiveDataStore.Store(ctx, doc)

	ref := references.Reference{Source: doc.Uri()}
	MakeReferenceSummary(ctx, nil, &ref, ActiveDataStore)

	if ref.Summary != "Test 123" {
		t.Errorf("Unexpected reference summary: %s", ref.Summary)
	}
}

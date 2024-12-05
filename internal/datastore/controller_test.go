package datastore

import (
	"context"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
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

	// refs := assertion.References()

	// if len(refs) != 2 {
	// 	t.Errorf("Did not find exactly 2 references in assertion: %d", len(refs))
	// }
	ref := references.Reference{
		Source:  assertion.Uri(),
		Target:  issuerUri,
		Summary: "",
	}

	target := references.Referenceable(assertion)
	MakeSummary(context.TODO(), &target, &ref, ActiveDataStore)

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

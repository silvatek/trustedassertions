package testdata

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/docs"
	ref "silvatek.uk/trustedassertions/internal/references"
)

const shorthandDoc = `<?xml version="1.0" encoding="UTF-8"?>
<document>
	<metadata>
		<title>Shorthand Loader Doc</title>
	</metadata>
	<section>
		<title>Claims</title>
		<paragraph>
			<span assertion="IsTrue 0.9">Isaac Newton was a scientist</span>
		</paragraph>
		<paragraph>
			<span assertion="IsFalse 0.8">Isaac Newton was French</span>
		</paragraph>
	</section>
</document>
`

func TestInitialInviteRolesInMemory(t *testing.T) {
	roles := initialInviteRoles(datastore.NewInMemoryDataStore())
	if !containsRole(roles, auth.RoleAuthor) || !containsRole(roles, auth.RoleAdministrator) {
		t.Errorf("in-memory invite roles = %v, want Author and Administrator", roles)
	}
}

func TestInitialInviteRolesFirestore(t *testing.T) {
	roles := initialInviteRoles(&datastore.FireStore{})
	if roles != nil {
		t.Errorf("firestore invite roles = %v, want nil", roles)
	}
}

func TestSetupTestDataKeepsExistingDocumentAssertions(t *testing.T) {
	ctx := context.Background()
	datastore.InitInMemoryDataStore()
	assertions.PublicKeyResolver = datastore.ActiveDataStore

	signer := datastore.CreateEntityWithKey(ctx, "Loader Tester")
	key, err := datastore.ActiveDataStore.FetchKey(signer)
	if err != nil {
		t.Fatalf("FetchKey: %v", err)
	}

	SetupTestData(ctx, "../../testdata", signer.String(), key)

	poc := fetchDocumentByQuery(t, ctx, "GL93J73C")
	if got := poc.Sections[0].Paragraphs[0].Spans[1].Assertion; !strings.HasPrefix(got, "hash://sha256/") {
		t.Errorf("testdoc1 assertion = %q, want existing hash URI", got)
	}
}

func TestLoadDocumentsCreatesAssertionsFromShorthand(t *testing.T) {
	ctx := context.Background()
	datastore.InitInMemoryDataStore()
	assertions.PublicKeyResolver = datastore.ActiveDataStore

	signer := datastore.CreateEntityWithKey(ctx, "Loader Tester")
	loadDocuments(ctx, writeShorthandDoc(t), signer)

	doc := fetchDocumentByQuery(t, ctx, "Shorthand Loader Doc")
	var sawTrue, sawFalse bool
	for _, sect := range doc.Sections {
		for _, para := range sect.Paragraphs {
			for _, span := range para.Spans {
				if span.Assertion == "" {
					continue
				}
				if !strings.HasPrefix(span.Assertion, "hash://sha256/") {
					t.Errorf("shorthand was not replaced: %q (%s)", span.Assertion, span.Body)
					continue
				}
				assertion, err := datastore.ActiveDataStore.FetchAssertion(ctx, ref.UriFromString(span.Assertion))
				if err != nil {
					t.Errorf("FetchAssertion %s: %v", span.Assertion, err)
					continue
				}
				switch assertion.Category {
				case string(assertions.IsTrue):
					sawTrue = true
				case string(assertions.IsFalse):
					sawFalse = true
				}
			}
		}
	}
	if !sawTrue {
		t.Error("expected an IsTrue assertion from shorthand XML")
	}
	if !sawFalse {
		t.Error("expected an IsFalse assertion from shorthand XML")
	}
}

func TestLoadDocumentsSignsWithoutDefaultEntity(t *testing.T) {
	ctx := context.Background()
	datastore.InitInMemoryDataStore()
	assertions.PublicKeyResolver = datastore.ActiveDataStore

	loadDocuments(ctx, writeShorthandDoc(t), documentSigner(ctx, ""))

	doc := fetchDocumentByQuery(t, ctx, "Shorthand Loader Doc")
	found := false
	for _, sect := range doc.Sections {
		for _, para := range sect.Paragraphs {
			for _, span := range para.Spans {
				if strings.HasPrefix(span.Assertion, "hash://sha256/") {
					if _, err := datastore.ActiveDataStore.FetchAssertion(ctx, ref.UriFromString(span.Assertion)); err != nil {
						t.Errorf("FetchAssertion: %v", err)
					}
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected shorthand assertions to be created without a default entity")
	}
}

func writeShorthandDoc(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shorthand.xml"), []byte(shorthandDoc), 0644); err != nil {
		t.Fatalf("write shorthand.xml: %v", err)
	}
	return dir
}

func fetchDocumentByQuery(t *testing.T, ctx context.Context, query string) *docs.Document {
	t.Helper()
	results, err := datastore.ActiveDataStore.Search(ctx, query)
	if err != nil {
		t.Fatalf("Search %q: %v", query, err)
	}
	for _, result := range results {
		doc, err := datastore.ActiveDataStore.FetchDocument(ctx, result.Uri)
		if err != nil {
			continue
		}
		return &doc
	}
	t.Fatalf("no document matched %q (%d results)", query, len(results))
	return nil
}

func containsRole(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}

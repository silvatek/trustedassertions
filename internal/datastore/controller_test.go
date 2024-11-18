package datastore

import (
	"context"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/references"
)

func TestMakeSummary(t *testing.T) {
	ctx := context.Background()
	InitInMemoryDataStore()
	assertions.PublicKeyResolver = ActiveDataStore

	issuerUri := CreateEntityWithKey(ctx, "Unit Tester")

	assertion, err := CreateStatementAndAssertion(ctx, "Test 123", issuerUri, 0.9)
	if err != nil {
		t.Error(err)
	}

	refs := assertion.References()

	if len(refs) != 2 {
		t.Errorf("Did not find exactly 2 references in assertion: %d", len(refs))
	}

	target := references.Referenceable(assertion)
	MakeSummary(context.TODO(), &target, &(refs[0]), ActiveDataStore)

	if refs[0].Summary != "Unit Tester claims that 'Test 123' is true" {
		t.Errorf("Unexpected assertion summary: %s", refs[0].Summary)
	}
}

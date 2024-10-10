package assertions

import (
	"strings"
	"testing"
)

func TestStatementUri(t *testing.T) {
	verifyStatementUri("", t)
	verifyStatementUri("T", t)
	verifyStatementUri("The world is flat", t)
}

func verifyStatementUri(content string, t *testing.T) {
	statement := Statement{content: content}
	uri := statement.Uri()

	if !strings.HasPrefix(uri.String(), "hash://sha256/") {
		t.Errorf("Statement URI does not have correct prefix: %s", uri)
	}

	if uri.Len() != 93 {
		t.Errorf("Statement URI is not correct length: %d", uri.Len())
	}
}

func TestStatementMetaData(t *testing.T) {
	statement := NewStatement("Test statement")

	if statement.Type() != "Statement" {
		t.Errorf("Unexpected statement type: %s", statement.Type())
	}

	if statement.Content() != "Test statement" {
		t.Errorf("Unexpected statement content: %s", statement.Content())
	}

	if statement.Summary() != "Test statement" {
		t.Errorf("Unexpected statement summary: %s", statement.Summary())
	}
}

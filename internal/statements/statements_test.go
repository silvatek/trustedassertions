package statements

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

	if statement.TextContent() != "Test statement" {
		t.Errorf("Unexpected statement TextContent: %s", statement.TextContent())
	}

	if len(statement.References()) != 0 {
		t.Errorf("Unexpected number of references: %d", len(statement.References()))
	}
}

func TestStatementSummary(t *testing.T) {
	short := NewStatement("Testing")
	if short.Summary() != "Testing" {
		t.Errorf("Incorrect short statement summary: %s", short.Summary())
	}

	long := NewStatement("Testing a really really really really really really really long statement")
	if long.Summary() != "Testing a really really really really really really real..." {
		t.Errorf("Incorrect long statement summary: %s", long.Summary())
	}
}

func TestParseContent(t *testing.T) {
	statement := Statement{}
	statement.ParseContent("test 1234")
	if statement.Content() != "test 1234" {
		t.Errorf("Unexpected parsed statement content: %s", statement.Content())
	}
}

func TestNormaliseNewlines(t *testing.T) {
	source := []byte("1\n2\r\n3\r")
	modified := NormalizeNewlines(source)
	if len(modified) != 6 {
		t.Errorf("Unexpected normalised text: %v", modified)
	}
}

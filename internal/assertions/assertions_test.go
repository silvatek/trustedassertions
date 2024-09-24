package assertions

import "testing"

func TestHashUri(t *testing.T) {
	uri := makeUri("Some content")
	if uri != "hash://sha256/9c6609fc5111405ea3f5bb3d1f6b5a5efd19a0cec53d85893fd96d265439cd5b" {
		t.Errorf("Unexpected hash content: " + string(uri))
	}
}

func TestSimpleAssertions(t *testing.T) {
	statements = make(map[HashUri]Statement)
	entities = make(map[HashUri]Entity)
	assertions = make(map[HashUri]Assertion)

	statement1 := Statement("Some content")
	statements[statementUri(statement1)] = statement1
}

package search

import "testing"

func TestStripPunctuation(t *testing.T) {
	data := map[string]string{
		"":          "",
		"word":      "word",
		"word.":     "word",
		"word,":     "word",
		"word;":     "word",
		"word?":     "word",
		"word.;.;?": "word",
		"'word'":    "word",
	}

	for input, expected := range data {
		output := stripPunctuation(input)
		if output != expected {
			t.Errorf("Unexpected output for %s : %s", input, output)
		}
	}
}

func TestSearchWords(t *testing.T) {
	data := map[string][]string{
		"":               {},
		" \t\n ":         {},
		"test":           {"test"},
		"Test":           {"test"},
		"Test Words":     {"test", "words"},
		"But nothing":    {"nothing"},
		"Universal jobs": {"universe", "jobs"},
		"Question? Mark": {"question", "mark"},
	}
	for input, expected := range data {
		output := SearchWords(input)
		if !equal(output, expected) {
			t.Errorf("Unexpected output for `%s`: %v", input, output)
		}
	}
}

func equal(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, val := range a {
		if b[i] != val {
			return false
		}
	}
	return true
}

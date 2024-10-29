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
		"":                {},
		" \t\n ":          {},
		"test":            {"test"},
		"Test":            {"test"},
		"Test Words":      {"test", "words"},
		"But nothing":     {"nothing"},
		"Universal jobs":  {"jobs", "universe"},
		"Question? Mark":  {"mark", "question"},
		"Question ? Mark": {"mark", "question"},
		"Red Blue Red":    {"blue", "red"},
	}
	for input, expected := range data {
		output := SearchWords(input)
		if !WordsEqual(output, expected) {
			t.Errorf("Unexpected output for `%s`: %v", input, output)
		}
	}
}

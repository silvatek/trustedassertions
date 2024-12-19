package search

import (
	"testing"
)

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

func TestWordsEqual(t *testing.T) {
	words0 := []string{"Red", "Cat"}
	words1 := []string{"Red", "Cat"}
	words2 := []string{"Cat"}
	words3 := []string{"Blue", "Cat"}

	if !WordsEqual(words0, words1) {
		t.Errorf("Same word lists should be equal")
	}
	if WordsEqual(words0, words2) {
		t.Errorf("Different sized word lists should not be equal")
	}
	if WordsEqual(words0, words3) {
		t.Errorf("Different content word lists should not be equal")
	}
}

package search

import (
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/umahmood/soundex"
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

func TestSoundexVector(t *testing.T) {
	words := []string{"universe", "universal", "anniversary", "see", "sea", "see?", "cease"}

	vec := make(firestore.Vector64, len(words))

	for i, word := range words {
		code := soundex.Code(word)
		term := float64(0)
		mult := float64(1)
		for n := 3; n >= 0; n-- {
			term = term + mult*float64(code[n])
			mult = mult * float64(256)
		}

		t.Errorf("%s (%s) -> %f", word, code, term)

		vec[i] = term
	}

	t.Errorf("Vector = %v", vec)

}

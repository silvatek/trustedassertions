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

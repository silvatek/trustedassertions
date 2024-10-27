package search

import "strings"

func SearchWords(text string) []string {
	searchWords := make([]string, 0)
	allWords := strings.Fields(text)
	for _, word := range allWords {
		word := strings.ToLower(word)
		word = stripPunctuation(word)
		if ignoredWord(word) {
			continue
		}
		word = wordRoot(word)
		searchWords = append(searchWords, word)
	}
	return searchWords
}

const punct = ".,?;:'\""

func stripPunctuation(word string) string {
	if word == "" {
		return ""
	}
	var stripped strings.Builder
	for _, wordRune := range word {
		skip := false
		for _, punctRune := range punct {
			if wordRune == punctRune {
				skip = true
				break
			}
		}
		if !skip {
			stripped.WriteRune(wordRune)
		}
	}
	return stripped.String()
}

var roots = map[string]string{
	"universal": "universe",
	"exports":   "exports",
}

func wordRoot(word string) string {
	root, ok := roots[word]
	if ok {
		return root
	}
	return word
}

var IgnoredWords = []string{"a", "an", "it", "is", "the", "and", "but"}

func ignoredWord(word string) bool {
	for _, w := range IgnoredWords {
		if word == w {
			return true
		}
	}
	return false
}

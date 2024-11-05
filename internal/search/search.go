package search

import (
	"sort"
	"strings"
)

func SearchWords(text string) []string {
	wordMap := make(map[string]bool)
	allWords := strings.Fields(text)
	for _, word := range allWords {
		word := strings.ToLower(word)
		word = stripPunctuation(word)
		if len(word) == 0 {
			continue
		}
		if ignoredWord(word) {
			continue
		}
		word = wordRoot(word)
		wordMap[word] = true
	}

	searchWords := make([]string, 0)
	for key, _ := range wordMap {
		searchWords = append(searchWords, key)
	}

	sort.Strings(searchWords)

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
	"exists":    "exist",
	"truths":    "truth",
	"start":     "begin",
	"seem":      "appear",
}

func wordRoot(word string) string {
	root, ok := roots[word]
	if ok {
		return root
	}
	return word
}

var IgnoredWords = []string{"a", "an", "as", "it", "is", "the", "this", "and", "but", "i", "we", "to"}

func ignoredWord(word string) bool {
	for _, w := range IgnoredWords {
		if word == w {
			return true
		}
	}
	return false
}

func WordsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, word := range a {
		if b[i] != word {
			return false
		}
	}
	return true
}

func WordsContains(wordList []string, words ...string) bool {
	matches := 0
	for _, w1 := range wordList {
		for _, w2 := range words {
			if w1 == w2 {
				matches++
			}
		}
	}
	return matches == len(words)
}

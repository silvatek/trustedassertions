package auth

import (
	_ "embed"
	"math/rand/v2"
	"regexp"
	"strings"
)

type Registration struct {
	Code     string   `json:"code" firestore:"Code"`
	Status   string   `json:"status" firestore:"Status"`
	UserName string   `json:"username" firestore:"UserName"`
	Roles    []string `json:"roles" firestore:"Roles"`
}

const InviteCodeWordCount = 4

//go:embed registration_code_words.txt
var inviteWordsFile string

var inviteWords = strings.Fields(inviteWordsFile)
var inviteWhitespace = regexp.MustCompile(`\s+`)

// NormalizeInviteCode prepares a code typed by a user: lowercase, trim
// leading and trailing whitespace, and collapse any run of whitespace to a
// single space, e.g. "  Oak\tTree  Blue   Sky " -> "oak tree blue sky".
func NormalizeInviteCode(code string) string {
	code = strings.ToLower(code)
	code = strings.TrimSpace(code)
	return inviteWhitespace.ReplaceAllString(code, " ")
}

// GenerateInviteCode picks four distinct words from the in-repo list.
func GenerateInviteCode() string {
	used := make(map[string]struct{}, InviteCodeWordCount)
	words := make([]string, 0, InviteCodeWordCount)
	for len(words) < InviteCodeWordCount {
		word := inviteWords[rand.IntN(len(inviteWords))]
		if _, ok := used[word]; ok {
			continue
		}
		used[word] = struct{}{}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

// InviteWordCount is the size of the curated word list (for tests).
func InviteWordCount() int {
	return len(inviteWords)
}

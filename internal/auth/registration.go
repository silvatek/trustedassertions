package auth

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
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

var inviteWords = parseInviteWords(inviteWordsFile)

func parseInviteWords(raw string) []string {
	lines := strings.Split(raw, "\n")
	words := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		word := strings.ToLower(strings.TrimSpace(line))
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		words = append(words, word)
	}
	return words
}

// NormalizeInviteCode strips whitespace and lowercases so spaced or mixed-case
// phrases match the stored registration document id.
func NormalizeInviteCode(code string) string {
	return strings.ToLower(strings.Join(strings.Fields(code), ""))
}

// GenerateInviteCode picks four distinct words from the in-repo list.
// display is the spaced phrase for reading aloud; normalized is the stored code.
func GenerateInviteCode() (display string, normalized string, err error) {
	if len(inviteWords) < InviteCodeWordCount {
		return "", "", fmt.Errorf("invite word list has %d words, need at least %d", len(inviteWords), InviteCodeWordCount)
	}
	chosen := make([]string, 0, InviteCodeWordCount)
	used := make(map[int]struct{}, InviteCodeWordCount)
	for len(chosen) < InviteCodeWordCount {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteWords))))
		if err != nil {
			return "", "", err
		}
		i := int(n.Int64())
		if _, ok := used[i]; ok {
			continue
		}
		used[i] = struct{}{}
		chosen = append(chosen, inviteWords[i])
	}
	display = strings.Join(chosen, " ")
	return display, NormalizeInviteCode(display), nil
}

// InviteWordCount is the size of the curated word list (for tests).
func InviteWordCount() int {
	return len(inviteWords)
}

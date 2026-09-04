package auth

import (
	_ "embed"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"
)

const InviteValidity = 7 * 24 * time.Hour

type Registration struct {
	Code        string    `json:"code" firestore:"Code"`
	Status      string    `json:"status" firestore:"Status"`
	UserName    string    `json:"username" firestore:"UserName"`
	Roles       []string  `json:"roles" firestore:"Roles"`
	CreatedAt   time.Time `json:"createdAt" firestore:"CreatedAt"`
	CreatedBy   string    `json:"createdBy" firestore:"CreatedBy"`
	CompletedAt time.Time `json:"completedAt" firestore:"CompletedAt"`
	ExpiresAt   time.Time `json:"expiresAt" firestore:"ExpiresAt"`
}

// NewPendingRegistration is a new invite: pending, created now (UTC),
// expiring after InviteValidity. Zero ExpiresAt on older stored docs means
// the code does not expire.
func NewPendingRegistration(code, createdBy string, roles []string) Registration {
	now := time.Now().UTC()
	return Registration{
		Code:      code,
		Status:    "Pending",
		Roles:     roles,
		CreatedAt: now,
		CreatedBy: createdBy,
		ExpiresAt: now.Add(InviteValidity),
	}
}

// IsExpired reports whether ExpiresAt is set and at-or-before the given time.
// A zero ExpiresAt is never expired (legacy records).
func (r Registration) IsExpired(at time.Time) bool {
	if r.ExpiresAt.IsZero() {
		return false
	}
	return !at.UTC().Before(r.ExpiresAt.UTC())
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

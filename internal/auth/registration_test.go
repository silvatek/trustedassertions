package auth

import (
	"strings"
	"testing"
)

func TestNormalizeInviteCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "spaced mixed case", input: "Oak Tree Blue Sky", want: "oaktreebluesky"},
		{name: "already normalized", input: "oaktreebluesky", want: "oaktreebluesky"},
		{name: "upper spaced", input: "OAK TREE BLUE SKY", want: "oaktreebluesky"},
		{name: "extra spaces", input: "  oak   tree blue  sky ", want: "oaktreebluesky"},
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
		{name: "keeps hyphen", input: "TESTCODE-1001", want: "testcode-1001"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeInviteCode(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeInviteCode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateInviteCode(t *testing.T) {
	if InviteWordCount() < 200 {
		t.Fatalf("invite word list too small: %d", InviteWordCount())
	}

	display, normalized, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode: %v", err)
	}

	words := strings.Fields(display)
	if len(words) != InviteCodeWordCount {
		t.Fatalf("display %q has %d words, want %d", display, len(words), InviteCodeWordCount)
	}
	if normalized != NormalizeInviteCode(display) {
		t.Errorf("normalized %q does not match display %q", normalized, display)
	}
	if strings.ContainsAny(normalized, " \t\n") {
		t.Errorf("normalized code still has whitespace: %q", normalized)
	}

	seen := make(map[string]struct{}, len(words))
	allowed := make(map[string]struct{}, InviteWordCount())
	for _, w := range inviteWords {
		allowed[w] = struct{}{}
	}
	for _, w := range words {
		if _, ok := allowed[w]; !ok {
			t.Errorf("word %q is not in the invite list", w)
		}
		if _, dup := seen[w]; dup {
			t.Errorf("duplicate word %q in %q", w, display)
		}
		seen[w] = struct{}{}
	}
}

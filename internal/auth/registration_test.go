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
		{name: "spaced mixed case", input: "Oak Tree Blue Sky", want: "oak tree blue sky"},
		{name: "already normalized", input: "oak tree blue sky", want: "oak tree blue sky"},
		{name: "upper spaced", input: "OAK TREE BLUE SKY", want: "oak tree blue sky"},
		{name: "extra spaces", input: "  oak   tree blue  sky ", want: "oak tree blue sky"},
		{name: "tabs and newlines", input: "oak\ttree\nblue\r\nsky", want: "oak tree blue sky"},
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
		{name: "single token", input: "ABC", want: "abc"},
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

	code := GenerateInviteCode()

	words := strings.Fields(code)
	if len(words) != InviteCodeWordCount {
		t.Fatalf("code %q has %d words, want %d", code, len(words), InviteCodeWordCount)
	}
	if code != strings.Join(words, " ") {
		t.Errorf("code should be words joined by single spaces: %q", code)
	}
	if code != strings.ToLower(code) {
		t.Errorf("code should be lowercase: %q", code)
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
			t.Errorf("duplicate word %q in %q", w, code)
		}
		seen[w] = struct{}{}
	}
}

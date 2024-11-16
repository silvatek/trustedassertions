package assertions

import (
	"crypto/sha256"

	. "silvatek.uk/trustedassertions/internal/references"
)

type Statement struct {
	uri     HashUri
	content string
}

func NewStatement(content string) *Statement {
	return &Statement{content: content}
}

func (s Statement) Uri() HashUri {
	if s.uri.IsEmpty() {
		hash := sha256.New()
		hash.Write([]byte(s.content))
		s.uri = MakeUriB(hash.Sum(nil), "statement")
	}
	return s.uri
}

func (s Statement) Type() string {
	return "Statement"
}

func (s Statement) Content() string {
	return s.content
}

func (s Statement) Summary() string {
	return s.content
}

func (s Statement) TextContent() string {
	return s.content
}

func (s Statement) References() []HashUri {
	return []HashUri{}
}

func (s *Statement) ParseContent(content string) error {
	s.content = content
	return nil
}

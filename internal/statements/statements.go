package statements

import (
	refs "silvatek.uk/trustedassertions/internal/references"
)

type Statement struct {
	uri     refs.HashUri
	content string
}

func NewStatement(content string) *Statement {
	return &Statement{content: content}
}

func (s Statement) Uri() refs.HashUri {
	if s.uri.IsEmpty() {
		s.uri = refs.UriFor(&s)
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

func (s Statement) References() []refs.HashUri {
	return []refs.HashUri{}
}

func (s *Statement) ParseContent(content string) error {
	s.content = content
	return nil
}

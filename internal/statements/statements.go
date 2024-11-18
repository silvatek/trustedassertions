package statements

import (
	"bytes"

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

// Returns a summary of the statement, up to 60 characters long.
func (s Statement) Summary() string {
	if len(s.content) < 60 {
		return s.content
	} else {
		return s.content[0:56] + "..."
	}

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

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}

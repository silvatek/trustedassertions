package assertions

import (
	"crypto/sha256"
	"fmt"
)

type Statement struct {
	uri     string
	content string
}

func NewStatement(content string) Statement {
	return Statement{content: content}
}

func (s *Statement) Uri() string {
	if s.uri == "" {
		hash := sha256.New()
		hash.Write([]byte(s.content))
		return fmt.Sprintf("hash://sha256/%x", hash.Sum(nil))
	}
	return s.uri
}

func (s *Statement) Type() string {
	return "Statement"
}

func (s *Statement) Content() string {
	return s.content
}

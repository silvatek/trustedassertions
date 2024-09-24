package assertions

import (
	"crypto/sha256"
	"encoding/hex"
)

type HashUri string
type Statement string

type Entity struct {
	publicKey  string
	commonName string
}

type Assertion struct {
}

var statements map[HashUri]Statement
var entities map[HashUri]Entity
var assertions map[HashUri]Assertion

func makeUri(text string) HashUri {
	var hash = sha256.Sum256([]byte(text))
	return HashUri("hash://sha256/" + hex.EncodeToString(hash[:]))
}

func statementUri(statement Statement) HashUri {
	return makeUri(string(statement))
}

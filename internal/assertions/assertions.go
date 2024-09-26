package assertions

import (
	"github.com/golang-jwt/jwt/v5"
)

type Statement struct {
	*jwt.RegisteredClaims
	Content string `json:content`
}

type Entity struct {
	*jwt.RegisteredClaims
	PublicKey  string `json:"key"`
	CommonName string `json:"name"`
}

type Assertion struct {
	*jwt.RegisteredClaims
}

package assertions

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/golang-jwt/jwt/v5"
)

var (
	PublicKey  rsa.PublicKey
	PrivateKey *rsa.PrivateKey
)

type AssertionClaims interface {
	Kind() string
}

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

func (s *Statement) Kind() string {
	return "Statement"
}

func (e *Entity) Kind() string {
	return "Entity"
}

func (a *Assertion) Kind() string {
	return "Assertion"
}

func InitKeyPair() {
	PrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	PublicKey = PrivateKey.PublicKey
}

func CreateJwt(value jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, value)

	ac, isAssertionClaims := value.(AssertionClaims)
	if isAssertionClaims {
		token.Header["kind"] = ac.Kind()
	}

	return token.SignedString(PrivateKey)
}

func verificationKey(token *jwt.Token) (interface{}, error) {
	return &PublicKey, nil
}

func ParseEntityJwt(token string) (Entity, error) {
	entity := Entity{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}
	_, err := jwt.ParseWithClaims(token, &entity, verificationKey)
	return entity, err
}

package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const DEFAULT_AUDIENCE = "trustedassertions:0.1/any"
const UNDEFINED_CATEGORY = "Undefined"

type Assertion struct {
	*jwt.RegisteredClaims
	Category   string  `json:"category,omitempty"`
	Confidence float32 `json:"confidence,omitempty"`
	Object     string  `json:"object,omitempty"`
	content    string  `json:"-"`
	uri        string  `json:"-"`
}

func NewAssertion(category string) Assertion {
	return Assertion{
		Category:   category,
		Confidence: 0.0,
		RegisteredClaims: &jwt.RegisteredClaims{
			Audience: []string{DEFAULT_AUDIENCE},
		},
	}
}

func verificationKey(token *jwt.Token) (interface{}, error) {
	return &PublicKey, nil
}

func ParseAssertionJwt(token string) (Assertion, error) {
	assertion := Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{},
		content:          token,
	}
	_, err := jwt.ParseWithClaims(token, &assertion, verificationKey)
	return assertion, err
}

func (a *Assertion) makeJwt() {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, a)
	a.content, _ = token.SignedString(PrivateKey)
}

func (a *Assertion) Uri() string {
	if a.uri == "" {
		if a.content == "" {
			a.makeJwt()
		}
		hash := sha256.New()
		hash.Write([]byte(a.Content()))
		a.uri = fmt.Sprintf("hash://sha256/%x", hash.Sum(nil))
	}
	return a.uri
}

func (a *Assertion) Type() string {
	return "Assertion"
}

func (a *Assertion) Content() string {
	if a.content == "" {
		a.makeJwt()
	}
	return a.content
}

func (a *Assertion) SetAssertingEntity(entity Entity) {
	a.RegisteredClaims.Issuer = entity.Uri()
}

//============================================//

var (
	PublicKey  rsa.PublicKey
	PrivateKey *rsa.PrivateKey
)

func InitKeyPair() {
	osKey := os.Getenv("PRV_KEY")
	if osKey == "" {
		log.Println("Generating new key pair")
		PrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		// Expects a base64 encoded RSA private key
		log.Println("Loading private key from environment")
		bytes, _ := base64.StdEncoding.DecodeString(osKey)
		PrivateKey, _ = x509.ParsePKCS1PrivateKey(bytes)
	}
	PublicKey = PrivateKey.PublicKey
}

func Base64Private() string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(PrivateKey))
}

//============================================//

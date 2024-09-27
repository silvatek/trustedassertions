package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Assertion struct {
	*jwt.RegisteredClaims
	Category string `json:"category"`
	content  string `json:"-"`
	sig      string `json:"-"`
}

func NewAssertion(category string) Assertion {
	return Assertion{Category: "{unknown}"}
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
	a.sig = jwtSignature(a.content)
}

func jwtSignature(token string) string {
	parts := strings.SplitN(token, ".", 2)
	return parts[0]
}

func (a *Assertion) Uri() string {
	if a.sig == "" {
		a.makeJwt()
	}
	return fmt.Sprintf("sig://jwt/%s", a.sig)
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

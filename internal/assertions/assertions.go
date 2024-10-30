package assertions

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	log "silvatek.uk/trustedassertions/internal/logging"
)

const DEFAULT_AUDIENCE = "trustedassertions:0.1/any"
const UNDEFINED_CATEGORY = "Undefined"

type Assertion struct {
	*jwt.RegisteredClaims
	Category   string  `json:"category,omitempty"`
	Confidence float32 `json:"confidence,omitempty"`
	Object     string  `json:"object,omitempty"`
	content    string  `json:"-"`
	uri        HashUri `json:"-"`
}

// Resolver used to fetch public keys for entities.
var PublicKeyResolver Resolver

func NewAssertion(category string) Assertion {
	return Assertion{
		Category:   category,
		Confidence: 0.0,
		RegisteredClaims: &jwt.RegisteredClaims{
			Audience: []string{DEFAULT_AUDIENCE},
		},
	}
}

// Returns the public key to be used to verify the specified JWT token.
// The token issuer should be the URI of an entity, and that entity is fetched using the PublicKeyResolver.
func verificationKey(token *jwt.Token) (interface{}, error) {
	entityUri, _ := token.Claims.GetIssuer()
	entity, err := PublicKeyResolver.FetchEntity(HashUri{uri: entityUri})
	return entity.PublicKey, err
}

func ParseAssertionJwt(token string) (Assertion, error) {
	template := Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}

	if token == "" {
		return template, errors.New("unable to parse empty JWT")
	}

	parsed, err := jwt.ParseWithClaims(token, &template, verificationKey)

	if assertion, ok := parsed.Claims.(*Assertion); ok && parsed.Valid {
		assertion.content = token
		return *assertion, err
	} else {
		return *assertion, errors.New("unable to parse JWT claims")
	}
}

func (a *Assertion) MakeJwt(privateKey *rsa.PrivateKey) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, a)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Errorf("Error creating signed JWT")
		return
	}
	a.content = signed
}

func (a *Assertion) Uri() HashUri {
	if a.uri.IsEmpty() {
		if a.content == "" {
			log.Errorf("Attempting to get URI for empty assertion content")
			return EMPTY_URI
		}
		hash := sha256.New()
		hash.Write([]byte(a.Content()))
		a.uri = MakeUriB(hash.Sum(nil), "assertion")
	}
	return a.uri
}

func (a *Assertion) Type() string {
	return "Assertion"
}

func (a *Assertion) Content() string {
	if a.content == "" {
		log.Errorf("Attempting to get URI for empty assertion content")
	}
	return a.content
}

func (a *Assertion) Summary() string {
	return a.Category + " Assertion"
}

func (a *Assertion) TextContent() string {
	return "" // Assertions aren't directly searchable
}

func (a *Assertion) SetAssertingEntity(entity Entity) {
	a.RegisteredClaims.Issuer = entity.Uri().String()
}

func GuessContentType(content string) string {
	if len(content) < 512 {
		// Both X509 certificates and JWTs signed by Entities are longer than 512 characters
		return "Statement"
	}
	if strings.HasPrefix(content, "-----BEGIN CERTIFICATE----") {
		// X509 certificates for Entities are self-describing
		return "Entity"
	}
	if strings.HasPrefix(content, "eyJ") {
		// Assertion JWTs start with bas64-encoded "{"
		return "Assertion"
	}
	if strings.HasPrefix(content, "<?xml") && strings.Contains(content, "<document>") {
		return "Document"
	}
	return "Statement"
}

func PrivateKeyToString(prvKey *rsa.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(prvKey))
}

func StringToPrivateKey(base64encoded string) *rsa.PrivateKey {
	bytes, _ := base64.StdEncoding.DecodeString(base64encoded)
	privateKey, err := x509.ParsePKCS1PrivateKey(bytes)
	if err != nil {
		log.Errorf("Error decoding key")
	}
	return privateKey
}

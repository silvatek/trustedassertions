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

type KeyFetcher interface {
	FetchKey(entityUri string) (string, error)
}

var ActiveKeyFetcher KeyFetcher

type EntityFetcher interface {
	FetchEntity(key string) (Entity, error)
}

var ActiveEntityFetcher EntityFetcher

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
// The token issuer should be the URI of an entity, and that entity is fetched from the data store.
func verificationKey(token *jwt.Token) (interface{}, error) {
	entityUri, _ := token.Claims.GetIssuer()
	entity, err := ActiveEntityFetcher.FetchEntity(entityUri)
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

func (a *Assertion) SetAssertingEntity(entity Entity) {
	a.RegisteredClaims.Issuer = entity.Uri().String()
}

func UriHash(uri string) string {
	hash := strings.TrimPrefix(uri, "hash://sha256/")
	queryIndex := strings.Index(hash, "?")
	if queryIndex > -1 {
		hash = hash[0:queryIndex]
	}
	return hash
}

func HashToUri(hash string, dataType string) string {
	uri := "hash://sha256/" + hash
	if dataType != "" {
		uri = uri + "?type=" + dataType
	}
	return uri
}

//============================================//

// // var (
// // 	PublicKey  rsa.PublicKey
// // 	PrivateKey *rsa.PrivateKey
// // )

// // func InitKeyPair(osKey string) {
// // 	if osKey == "" {
// // 		log.Info("Generating new key pair")
// // 		PrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
// // 	} else {
// // 		// Expects a base64 encoded RSA private key
// // 		log.Info("Loading private key from environment")
// // 		PrivateKey = PrivateBase64(osKey)
// // 	}
// // 	PublicKey = PrivateKey.PublicKey
// // }

// func Base64Private() string {
// 	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(PrivateKey))
// }

func DecodePrivateKey(prvKey *rsa.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(prvKey))
}

func EncodePrivateKey(base64encoded string) *rsa.PrivateKey {
	bytes, _ := base64.StdEncoding.DecodeString(base64encoded)
	privateKey, err := x509.ParsePKCS1PrivateKey(bytes)
	if err != nil {
		log.Errorf("Error decoding key")
	}
	return privateKey
}

//============================================//

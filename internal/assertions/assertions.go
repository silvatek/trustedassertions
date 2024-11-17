package assertions

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	refs "silvatek.uk/trustedassertions/internal/references"
)

const DEFAULT_AUDIENCE = "trustedassertions:0.1/any"
const UNDEFINED_CATEGORY = "Undefined"

type Assertion struct {
	*jwt.RegisteredClaims
	Category   string       `json:"category,omitempty"`
	Confidence float32      `json:"confidence,omitempty"`
	Object     string       `json:"object,omitempty"`
	content    string       `json:"-"`
	uri        refs.HashUri `json:"-"`
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
	entity, err := PublicKeyResolver.FetchEntity(context.Background(), refs.UriFromString(entityUri))
	return entity.PublicKey, err
}

func (a *Assertion) ParseContent(content string) error {
	a.content = content

	if content == "" {
		return errors.New("unable to parse empty JWT")
	}

	a.RegisteredClaims = &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(content, a, verificationKey)

	return err
}

func ParseAssertionJwt(token string) (Assertion, error) {
	template := Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}

	if token == "" {
		return template, errors.New("unable to parse empty JWT")
	}

	parsed, err := jwt.ParseWithClaims(token, &template, verificationKey)
	if err != nil {
		return template, err
	}

	if assertion, ok := parsed.Claims.(*Assertion); ok && parsed.Valid {
		assertion.content = token
		return *assertion, nil
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

func (a *Assertion) Uri() refs.HashUri {
	if a.uri.IsEmpty() {
		if a.content == "" {
			log.Errorf("Attempting to get URI for empty assertion content")
			return refs.ERROR_URI
		}
		a.uri = refs.UriFor(a)
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

func (a Assertion) References() []refs.Reference {
	references := make([]refs.Reference, 0)
	if a.RegisteredClaims.Issuer != "" {
		reference := refs.Reference{
			Source:  a.Uri(),
			Target:  refs.UriFromString(a.RegisteredClaims.Issuer),
			Summary: "Unspecified issuer reference",
		}
		references = append(references, reference)
	}
	if a.RegisteredClaims.Subject != "" {
		reference := refs.Reference{
			Source:  a.Uri(),
			Target:  refs.UriFromString(a.RegisteredClaims.Subject),
			Summary: "Unspecified subject reference",
		}
		references = append(references, reference)
	}
	return references
}

func (a *Assertion) SetAssertingEntity(entity entities.Entity) {
	a.RegisteredClaims.Issuer = entity.Uri().String()
}

func CategoryDescription(category string, language string) string {
	if strings.HasPrefix(language, "en") {
		switch category {
		case "IsTrue":
			return "is true"
		case "IsFalse":
			return "is false"
		default:
			return category
		}

	} else {
		return category
	}
}

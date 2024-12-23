package assertions

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/references"
)

const DEFAULT_AUDIENCE = "trustedassertions:0.1/any"
const UNDEFINED_CATEGORY = "Undefined"

type Assertion struct {
	*jwt.RegisteredClaims
	Category   string             `json:"category,omitempty"`
	Confidence float32            `json:"confidence,omitempty"`
	Object     string             `json:"object,omitempty"`
	content    string             `json:"-"`
	uri        references.HashUri `json:"-"`
	summary    string             `json:"-"`
}

type AssertionType string

const (
	IsTrue  AssertionType = "IsTrue"
	IsFalse AssertionType = "IsFalse"
	Unknown AssertionType = "Unknown"
)

var AssertionTypes = []AssertionType{IsTrue, IsFalse}

func (at AssertionType) String() string {
	return string(at)
}

func AssertionTypeOf(s string) AssertionType {
	for _, at := range AssertionTypes {
		if at.String() == s {
			return at
		}
	}
	return Unknown
}

// Resolver used to fetch public keys for entities.
var PublicKeyResolver Resolver

var log = logging.GetLogger("assertions")

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
	entity, err := PublicKeyResolver.FetchEntity(context.Background(), references.UriFromString(entityUri))
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
	assertion := Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}

	if token == "" {
		return assertion, errors.New("unable to parse empty JWT")
	}

	err := assertion.ParseContent(token)

	return assertion, err

	// parsed, err := jwt.ParseWithClaims(token, &assertion, verificationKey)
	// if err != nil {
	// 	return assertion, err
	// }

	// if assertion, ok := parsed.Claims.(*Assertion); ok && parsed.Valid {
	// 	assertion.content = token
	// 	return *assertion, nil
	// } else {
	// 	return *assertion, errors.New("unable to parse JWT claims")
	// }
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

func (a *Assertion) Uri() references.HashUri {
	if a.uri.IsEmpty() {
		if a.content == "" {
			log.Errorf("Attempting to get URI for empty assertion content")
			return references.ERROR_URI
		}
		a.uri = references.UriFor(a)
	}
	return a.uri
}

func (a *Assertion) Type() string {
	return "Assertion"
}

func (a *Assertion) Content() string {
	return a.content
}

func (a *Assertion) Summary() string {
	if a.summary == "" {
		a.summary = a.Category + " assertion"
	}
	return a.summary
}

func (a *Assertion) SetSummary(summary string) {
	a.summary = summary
}

func (a *Assertion) TextContent() string {
	return "" // Assertions aren't directly searchable
}

func (a Assertion) References() []references.HashUri {
	refs := make([]references.HashUri, 0)
	if a.RegisteredClaims.Issuer != "" {
		refs = append(refs, references.UriFromString(a.RegisteredClaims.Issuer))
	}
	if a.RegisteredClaims.Subject != "" {
		refs = append(refs, references.UriFromString(a.RegisteredClaims.Subject))
	}
	return refs
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

func SummariseAssertion(ctx context.Context, assertion Assertion, cache references.ReferenceMap, resolver Resolver) string {
	if cache == nil {
		cache = make(references.ReferenceMap)
	}

	var issuerName string

	issuerUri := references.UriFromString(assertion.Issuer)
	cached, found := cache[issuerUri]
	if found {
		issuerName = cached.Summary()
	} else {

		entity, _ := resolver.FetchEntity(ctx, issuerUri)
		issuerName = entity.Summary()
	}

	var subjectSummary string

	subjectUri := references.UriFromString(assertion.Subject)
	cached, found = cache[subjectUri]
	if found {
		subjectSummary = cached.Summary()
	} else {
		subject, _ := resolver.FetchStatement(ctx, subjectUri)
		subjectSummary = subject.Summary()
	}

	return fmt.Sprintf("%s claims that '%s' %s", issuerName, subjectSummary, CategoryDescription(assertion.Category, "en"))
}

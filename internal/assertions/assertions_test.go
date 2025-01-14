package assertions

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/logging"
	. "silvatek.uk/trustedassertions/internal/references"
	refs "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

func TestJwtSymmetric(t *testing.T) {
	// This approach isn't used, but keeping the code here for future reference.
	key := make([]byte, 10)
	rand.Reader.Read(key)

	keyStr := base64.StdEncoding.EncodeToString([]byte(key))
	t.Logf("Signing key: %s  %v", keyStr, key)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": "issuer",
			"sub": "entity",
		})
	signed, _ := token.SignedString(key)
	t.Logf("Signed token: %s", signed)

	parsedToken, _ := jwt.Parse(signed,
		func(token *jwt.Token) (interface{}, error) {
			return key, nil
		})

	t.Logf("Token header: %v", parsedToken.Header)
	t.Logf("Token Claims: %v", parsedToken.Claims)
}

func TestJwtAsymmetric(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := privateKey.PublicKey

	t.Log("Keypair generated")

	// Create a new JWT token
	assertion := &Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:  "your-issuer",
			Subject: "hash://sha256/1234",
		},
		Category: "IsTrue",
	}

	t.Log(assertion)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, assertion)

	// Sign the token with the private key
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
	}

	t.Logf("Signed Token: %v", signedToken)

	assertion2 := &Assertion{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}

	// Parse the token with the public key
	parsedToken, err := jwt.ParseWithClaims(signedToken, assertion2, func(token *jwt.Token) (interface{}, error) {
		return &publicKey, nil
	})
	if err != nil {
		t.Error(err)
	}

	if claims, ok := parsedToken.Claims.(*Assertion); ok && parsedToken.Valid {
		t.Logf("Subject: %s", claims.Subject)
		t.Logf("Category: %s", claims.Category)
	} else {
		t.Fail()
	}
}

type TestResolver struct {
	entity    entities.Entity
	statement statements.Statement
	NullResolver
}

func (r TestResolver) FetchEntity(ctx context.Context, key HashUri) (entities.Entity, error) {
	return r.entity, nil
}

func (r TestResolver) FetchStatement(ctx context.Context, key HashUri) (statements.Statement, error) {
	return r.statement, nil
}

func TestAssertionClaims(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	entity := entities.NewEntity("Test entity", *big.NewInt(123456))
	entity.MakeCertificate(privateKey)

	PublicKeyResolver = TestResolver{entity: entity} // TestEntityFetcher{entity: entity}

	assertion1 := NewAssertion("IsFalse")
	assertion1.Subject = "hash://sha256/12345678"
	assertion1.SetAssertingEntity(entity)
	assertion1.MakeJwt(privateKey)

	token := assertion1.Content()

	t.Logf("Assertion JWT = %s", token)

	assertion2, err := ParseAssertionJwt(token)
	t.Log(assertion2)
	t.Log(assertion2.RegisteredClaims)
	if err != nil {
		t.Errorf("Error parsing JWT: %v", err)
	}
	if assertion2.Subject != assertion1.Subject {
		t.Errorf("Subject does not match: %s", assertion2.Subject)
	}
	if assertion2.Audience[0] != DEFAULT_AUDIENCE {
		t.Error("No default audience found")
	}
}

func TestDecodePrivateKey(t *testing.T) {
	content := "MIIEpAIBAAKCAQEAr3xk/tl8uNGuxgZgCT72gDJaEdwDRCZC/hChe9tCHXy8fHNK1IvH2BtTDst/fKeKdHWvMFiPJuYhz6jIDiz3IlH7xDXiFOTYj55vSqfnv6D0JUwHqlIpBr/8EuuITIq8RJ3iXHJ6YLbBUPNWW0kXyEwTEXFVaaPipjGdjMtrHBgOIEiqS7Q+zkyNdxuhyQ0+zackC/Io/794L9KMC+TEL5wCvcmmmf/AE2VjsRqAn3Tq8qfp8B8/Oukotp7L4l1QY+VTaW7Xs/j2p9YKGzIVUOqy3htemCT4CczlndrkWib9hqowk/9yHLXtdsZflC2BGQG0+Qb4XMdWKNEEWccolQIDAQABAoIBAGWetGF8EiR4kCvuPyi7hEVCYzQaYu3Q9lRnzwuJMaUfuYqbvQaOF2EGmbdkkmPeJWhBSfzGG8eb1pKJG6hR057VOOUrisssplesmKfzyVkH3LnIaFvyDf3xqQhPynMAl/toLk/4nvNogVPeRfDAx/veSeB878gn9jTlYGXK2jC+q87hLQCYC1ynJ6IQQOQS5OXu6nZOf/bc6YQ7gUG35KT/hkSmFEYAu/y3pNGsRPYJcIbh23PwW0xcz9XQqddH0sOxwwevxuQWHEw38vE/YDRVaw7lXh77iRSiasRlQeeS3NjRSx00SJ+jVYwb0XapvrS/gggGB5TL4hDJhugARjUCgYEAz4iR35A8jZfYO7z4tBQukyo9dGGmh5JD1xmiDiCDjdMaGxQq85tDvS3s0jf60CMba0S5h48kT4UYmf6Ui72LZL7q6PHAJyih1It1VCTKG0pGlFCGRCY0/LMobkw1f51ZEyNRSsGgXswk+DezH53JqKhnHvSeOFcSjJ4FQnY7vTcCgYEA2Hfa3iyrv3Rf/HtzbhzHkTJtIWgyNiha/54h1JyTy3H9qCU5DRn3Uy3COBzieeDnFXZDr6Z+ik5wQ3mUaIXj/dH3pu53RWfiw2osJEcbj2T3bk2vkOGCz8EeELNGK+MiU5ETByQxLhFr6QU+QyZc5t2Anf5dYTi9ZUBvkywjjpMCgYBSd1hP2AbX1PDNvCevlx1yySQmbO85i/t9K+hjaLQd1TbYb8kpiBcAw5EJb8kwj+LDW0nF/jFVj/PYrXrllGohnGPIMNhENzcnOEtlJkFRWtB0+xJ/XhdMGv0D5zCTBzlwC2awKATL5p8CK0/4TkDlzhU8DcQZazApxFkesdDHPQKBgQDCZPAScX9TEclZTevdSM8XX3eNdqsQ47DEuVecPXikTRwEMllHoLfw5Ljz90yTMxuStIAYb6ZXwhUjIz3Zl9OlDzgdmy1VEPQdlW1KrujbH0rsrasqqrn0pHLBgJ1VsEYVUcUKtr/LpS2JN4Iwf3UShnyIZfOp6XB8Sx9nxU2xLQKBgQC/8Qvm7ehzlgW4gpAdtU3fgDJ056FWNuC1r7DlYxaesPtij/u81U3G3XHiROBD06SYzBTDCi/bjAwQUakmqtveMD1yFmBg1gle0YP73AscApfch0aG38NY37XYSo93IgxcNunc32lHv69xiHUFR7p2tbTdyf7/BPgFRa3NLYdWxQ=="

	bytes, _ := base64.StdEncoding.DecodeString(content)

	privateKey, _ := x509.ParsePKCS1PrivateKey(bytes)
	publicKey := privateKey.PublicKey

	t.Logf("Public key: %v", publicKey)

	// Create a new JWT token
	claims := &jwt.RegisteredClaims{
		Issuer: "your-issuer",
	}

	t.Logf("Claims: %v", claims)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign the token with the private key
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
	}

	t.Logf("Signed Token: %v", signedToken)

	// Parse the token with the public key
	parsedToken, err := jwt.ParseWithClaims(signedToken, claims, func(token *jwt.Token) (interface{}, error) {
		return &publicKey, nil
	})
	if err != nil {
		t.Errorf("Error parsing token: %v", err)
	}
	t.Logf("Parsed claims: %v", parsedToken.Claims)

	parsedIssuer, _ := parsedToken.Claims.GetIssuer()
	if parsedIssuer != "your-issuer" {
		t.Errorf("Unexpected issuer: %s", parsedIssuer)
	}
}

func TestKeyStringRoundTrip(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	s := entities.PrivateKeyToString(privateKey)

	key2 := entities.PrivateKeyFromString(s)

	if !key2.Equal(privateKey) {
		t.Error("Private key did not verify after round-trip through string")
	}
}

func TestAssertionJwt(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	a := NewAssertion("Simple")
	a.MakeJwt(privateKey)
	t.Log(a.Uri())
	t.Log(a.Content())

	if !strings.HasPrefix(a.Uri().String(), "hash://sha256/") {
		t.Errorf("Bad URI prefix: %s", a.Uri())
	}

	if a.Type() != "Assertion" {
		t.Errorf("Unexpected assertion type: %s", a.Type())
	}
	if a.Summary() != "Simple assertion" {
		t.Errorf("Unexpected assertion summary: %s", a.Summary())
	}

}

func TestUrlEncode(t *testing.T) {
	s := url.PathEscape("hash://sha256/123456")
	if s != "hash:%2F%2Fsha256%2F123456" {
		t.Errorf("Unexpected escaped URI: %v", s)
	}

	s1, err := url.PathUnescape(s)
	if err != nil {
		t.Errorf("Error unescaping URI: %v", err)
	}

	if s1 != "hash://sha256/123456" {
		t.Errorf("Error in round-tripped URI: %s", s1)
	}
}

func TestGuessContentType(t *testing.T) {
	buf := make([]byte, 512)
	for i := range buf {
		buf[i] = 'X'
	}
	padding := string(buf)

	data := map[string]string{
		"Testing":                              "Statement",
		"-----BEGIN CERTIFICATE----" + padding: "Entity",
		"eyJ" + padding:                        "Assertion",
		"<?xml version=\"1.0\"?><document>":    "Document",
		"Some text" + padding:                  "Statement",
		padding:                                "Statement",
		padding[0:511]:                         "Statement",
		"eyJ":                                  "Statement",
		"":                                     "Statement",
	}

	for input, expected := range data {
		output := GuessContentType(input)
		if output != expected {
			t.Errorf("Unexpected type for `%s`: %s", input, output)
		}
	}
}

func TestMakeReferenceable(t *testing.T) {
	testdata := map[string]string{
		"statement": "*statements.Statement",
		"Statement": "*statements.Statement",
		"entity":    "*entities.Entity",
		"assertion": "*assertions.Assertion",
		"document":  "*docs.Document",
	}
	for key, typeName := range testdata {
		item := NewReferenceable(key)
		itemType := reflect.TypeOf(item)
		if itemType.String() != typeName {
			t.Errorf("Unexpected type for %s: %v", key, itemType)
		}
	}
}

func TestParseContent(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := entities.NewEntity("Signer", *big.NewInt(1234))
	entity.MakeCertificate(privateKey)

	PublicKeyResolver = TestResolver{
		entity: entity,
	}

	assertion := NewAssertion("IsTrue")
	assertion.SetAssertingEntity(entity)
	assertion.Subject = "hash://sha256/123456"

	assertion.MakeJwt(privateKey)

	content := assertion.content

	assertion2 := Assertion{}
	err := assertion2.ParseContent(content)

	if err != nil {
		t.Errorf("Error parsing content: %v", err)
	}
	if assertion.Category != "IsTrue" {
		t.Errorf("Unexpected assertion categgory: %s", assertion2.Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	assertion := Assertion{}
	err := assertion.ParseContent("")
	if err == nil {
		t.Error("Should not be able to parse empty content")
	}
}

func TestAssertiontextContent(t *testing.T) {
	assertion := Assertion{}
	assertion.SetSummary("Testing")
	if assertion.TextContent() != "" {
		t.Errorf("Assertions should not have text content")
	}
}

func TestAssertionUri(t *testing.T) {
	a := Assertion{}
	if a.Uri() != refs.ERROR_URI {
		t.Error("Empty assertion does not have error URI")
	}
	a.content = "Testing"
	if a.Uri() == refs.ERROR_URI {
		t.Error("Non-empty assertion does has error URI")
	}
}

func TestAssertionCategory(t *testing.T) {
	testdata := map[string]string{
		"IsTrue":  "is true",
		"IsFalse": "is false",
		"IsSame":  "IsSame",
		"":        "",
	}
	for key, expected := range testdata {
		actual := CategoryDescription(key, "en")
		if actual != expected {
			t.Errorf("Unexpected category description for %s: %s", key, actual)
		}
	}

	if CategoryDescription("IsTrue", "fr") != "IsTrue" {
		t.Error("Non-English description does not equal category")
	}
}

func TestNullResolver(t *testing.T) {
	ctx := context.Background()
	r := NullResolver{}

	var err error

	_, err = r.FetchAssertion(ctx, ERROR_URI)
	assertErrorNotImplemented(err, t)

	_, err = r.FetchStatement(ctx, ERROR_URI)
	assertErrorNotImplemented(err, t)

	_, err = r.FetchEntity(ctx, ERROR_URI)
	assertErrorNotImplemented(err, t)

	_, err = r.FetchDocument(ctx, ERROR_URI)
	assertErrorNotImplemented(err, t)

	_, err = r.FetchKey(ERROR_URI)
	assertErrorNotImplemented(err, t)

	_, err = r.FetchRefs(ctx, ERROR_URI)
	assertErrorNotImplemented(err, t)

}

func assertErrorNotImplemented(err error, t *testing.T) {
	if err != ErrNotImplemented {
		t.Error("Expected ErrNotImplemented")
	}
}

func TestNewBadReferenceable(t *testing.T) {
	if NewReferenceable("broken") != nil {
		t.Error("New broken referenceable was not null")
	}
}

func TestSummariseAssertion(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := entities.NewEntity("Tester", *big.NewInt(1234))
	entity.MakeCertificate(privateKey)

	statement := *statements.NewStatement("Some statement")
	resolver := TestResolver{entity: entity, statement: statement}

	assertion := NewAssertion("IsTrue")
	assertion.SetAssertingEntity(entity)
	assertion.Subject = statement.Uri().String()

	summary := SummariseAssertion(context.Background(), assertion, nil, resolver)

	if summary != "Tester claims that 'Some statement' is true" {
		t.Errorf("Unexpected assertion summary: %s", summary)
	}

	cache := make(ReferenceMap)
	cache[statement.Uri()] = &statement
	cache[entity.Uri()] = &entity

	summary = SummariseAssertion(context.Background(), assertion, cache, resolver)

	if summary != "Tester claims that 'Some statement' is true" {
		t.Errorf("Unexpected assertion summary (2): %s", summary)
	}

}

func TestAssertionReferences(t *testing.T) {
	entity := entities.NewEntity("Tester", *big.NewInt(0))
	statement := statements.NewStatement("Some statement")

	assertion := NewAssertion("IsTrue")
	assertion.SetAssertingEntity(entity)
	assertion.Subject = statement.Uri().String()

	refs := assertion.References()

	if len(refs) != 2 {
		t.Errorf("Unexpected number of references: %d", len(refs))
	}
}

func TestMakeJwtError(t *testing.T) {
	var logs bytes.Buffer
	log = logging.Logger{
		Writer: &logs,
	}

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKey.E = 0

	assertion := NewAssertion("IsTrue")
	assertion.MakeJwt(privateKey)

	logged := logs.String()
	if !strings.Contains(logged, "Error creating signed JWT") {
		t.Errorf("Unexpected log content: %s", logged)
	}

	if assertion.content != "" {
		t.Errorf("Unexpected assertion content after failed MakeJwt: %s", assertion.content)
	}
}

func TestParseEmptyJwt(t *testing.T) {
	_, err := ParseAssertionJwt("")

	if err == nil {
		t.Errorf("Parsing empty JWT did not result in an error")
	}
}

func TestAssertionTypes(t *testing.T) {
	if AssertionTypeOf("IsTrue") != IsTrue {
		t.Error("IsTrue assertion type not correctly identified")
	}
	if AssertionTypeOf("Something completely different") != Unknown {
		t.Error("Unknown assertion type not correctly handled")
	}
}

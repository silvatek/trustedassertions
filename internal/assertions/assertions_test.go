package assertions

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJwtSymmetric(t *testing.T) {
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

// func TestJwtAsymmetric(t *testing.T) {
// 	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
// 	publicKey := privateKey.PublicKey

// 	t.Log("Keypair generated")

// 	// Create a new JWT token
// 	claims := &Entity{
// 		RegisteredClaims: &jwt.RegisteredClaims{
// 			Issuer: "your-issuer",
// 		},
// 		CommonName: "John Doe",
// 	}

// 	t.Log(claims)

// 	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

// 	// Sign the token with the private key
// 	signedToken, err := token.SignedString(privateKey)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	t.Logf("Signed Token: %v", signedToken)

// 	// Parse the token with the public key
// 	parsedToken, err := jwt.ParseWithClaims(signedToken, claims, func(token *jwt.Token) (interface{}, error) {
// 		return &publicKey, nil
// 	})
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if claims, ok := parsedToken.Claims.(*Entity); ok && parsedToken.Valid {
// 		t.Logf("Name: %s", claims.CommonName)
// 	} else {
// 		t.Fail()
// 	}
// }

// func TestAssertionClaims(t *testing.T) {
// 	InitKeyPair()

// 	entity1 := &Entity{
// 		CommonName: "John Smith",
// 	}

// 	token, err := CreateJwt(entity1)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	t.Error(token)

// 	entity2, err := ParseEntityJwt(token)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if entity2.CommonName != entity1.CommonName {
// 		t.Errorf("Common name does not match: '%s' != '%s'", entity2.CommonName, entity1.CommonName)
// 	}
// }

func TestEntityUri(t *testing.T) {
	entity := NewEntity("Tester")
	entity.AssignSerialNum()
	uri := entity.Uri()
	if uri == "" {
		t.Error("Empty entity URI")
	}
	if !strings.HasPrefix(uri, "cert://x509/") {
		t.Errorf("Entity URI does not have correct prefix: %s", uri)
	}
}

func TestCreateCertificate(t *testing.T) {
	InitKeyPair()

	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)
	template := x509.Certificate{
		SerialNumber:          serNum,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{Organization: []string{"Testing"}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 2),
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &PublicKey, PrivateKey)
	if err != nil {
		t.Error(err)
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert}
	certPEM := pem.EncodeToMemory(&b)
	if string(certPEM) == "" {
		t.Fail()
	}

}

func TestEntityCertificate(t *testing.T) {
	InitKeyPair()

	entity := &Entity{
		CommonName: "John Smith",
	}

	entity.MakeCertificate()

	if entity.Certificate == "" {
		t.Error(entity)
	}
}

func TestStatementUri(t *testing.T) {
	verifyStatementUri("", t)
	verifyStatementUri("T", t)
	verifyStatementUri("The world is flat", t)
}

func verifyStatementUri(content string, t *testing.T) {
	statement := Statement{content: content}
	uri := statement.Uri()
	if !strings.HasPrefix(uri, "hash://sha256/") {
		t.Errorf("Statement URI does not have correct prefix: %s", uri)
	}
	if len(uri) != 78 {
		t.Errorf("Statement URI is not correct length: %d", len(uri))
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

	t.Fail()
}

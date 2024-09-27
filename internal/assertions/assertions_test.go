package assertions

import (
	"crypto/rand"
	"crypto/rsa"
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

func TestJwtAsymmetric(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := privateKey.PublicKey

	t.Log("Keypair generated")

	// Create a new JWT token
	claims := &Entity{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer: "your-issuer",
		},
		CommonName: "John Doe",
	}

	t.Log(claims)

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
		t.Error(err)
	}

	if claims, ok := parsedToken.Claims.(*Entity); ok && parsedToken.Valid {
		t.Logf("Name: %s", claims.CommonName)
	} else {
		t.Fail()
	}
}

func TestAssertionClaims(t *testing.T) {
	InitKeyPair()

	entity1 := &Entity{
		CommonName: "John Smith",
	}

	token, err := CreateJwt(entity1)
	if err != nil {
		t.Error(err)
	}

	t.Error(token)

	entity2, err := ParseEntityJwt(token)
	if err != nil {
		t.Error(err)
	}

	if entity2.CommonName != entity1.CommonName {
		t.Errorf("Common name does not match: '%s' != '%s'", entity2.CommonName, entity1.CommonName)
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

	MakeEntityCertificate(entity)

	if entity.Certificate == "" {
		t.Error(entity)
	}

}

func TestStatementHash(t *testing.T) {
	verifyStatementUri("", t)
	verifyStatementUri("T", t)
	verifyStatementUri("The world is flat", t)
}

func verifyStatementUri(statement string, t *testing.T) {
	uri := StatementUri(statement)
	if !strings.HasPrefix(uri, "ash://sha256/") {
		t.Errorf("Statement URI does not have correct prefix: %s", uri)
	}
	if len(uri) != 78 {
		t.Errorf("Statement URI is not correct length: %d", len(uri))
	}

}

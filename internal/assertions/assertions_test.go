package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

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

	//t.Fail()
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
		PublicKey:  publicKey.N.String(),
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
		t.Logf("Key: %s", claims.PublicKey)
	} else {
		t.Fail()
	}
	t.Fail()
}

func TestAssertionClaims(t *testing.T) {
	IntiKeyPair()

	entity1 := &Entity{
		CommonName: "John Smith",
		PublicKey:  PublicKey.N.String(),
	}

	token, err := CreateJwt(entity1)
	if err != nil {
		t.Error(err)
	}

	t.Log(token)

	entity2, err := ParseEntityJwt(token)
	if err != nil {
		t.Error(err)
	}

	if entity2.CommonName != entity1.CommonName {
		t.Errorf("Common name does not match: '%s' != '%s'", entity2.CommonName, entity1.CommonName)
	}
}

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

// Define a custom claim struct to include additional information
type CustomClaims struct {
	*jwt.RegisteredClaims
	Name string `json:"name"`
	Role string `json:"role"`
}

func TestJwtAsymmetric(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := privateKey.PublicKey

	t.Log("Keypair generated")

	// Create a new JWT token
	claims := &CustomClaims{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer: "your-issuer",
		},
		Name: "John Doe",
		Role: "admin",
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

	if claims, ok := parsedToken.Claims.(*CustomClaims); ok && parsedToken.Valid {
		t.Logf("Name: %s", claims.Name)
		t.Logf("Role: %s", claims.Role)
	} else {
		t.Fail()
	}
}

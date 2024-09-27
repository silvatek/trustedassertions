package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	PublicKey  rsa.PublicKey
	PrivateKey *rsa.PrivateKey
)

type AssertionClaims interface {
	Kind() string
}

type Statement struct {
	*jwt.RegisteredClaims
	Content string `json:"content"`
}

type Entity struct {
	*jwt.RegisteredClaims
	CommonName  string `json:"name"`
	Certificate string `json:"cert"`
}

type Assertion struct {
	*jwt.RegisteredClaims
}

type Reference struct {
	target  string
	refType string
	source  string
}

func (s *Statement) Kind() string {
	return "Statement"
}

func (e *Entity) Kind() string {
	return "Entity"
}

func (a *Assertion) Kind() string {
	return "Assertion"
}

func InitKeyPair() {
	PrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	PublicKey = PrivateKey.PublicKey
}

func CreateJwt(value jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, value)

	ac, isAssertionClaims := value.(AssertionClaims)
	if isAssertionClaims {
		token.Header["kind"] = ac.Kind()
	}

	return token.SignedString(PrivateKey)
}

func verificationKey(token *jwt.Token) (interface{}, error) {
	return &PublicKey, nil
}

func ParseEntityJwt(token string) (Entity, error) {
	entity := Entity{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}
	_, err := jwt.ParseWithClaims(token, &entity, verificationKey)
	return entity, err
}

func MakeEntityCertificate(entity *Entity) {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)
	template := x509.Certificate{
		SerialNumber:          serNum,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{Organization: []string{entity.CommonName}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 2),
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &PublicKey, PrivateKey)
	if err != nil {
		return
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert}
	entity.Certificate = string(pem.EncodeToMemory(&b))
}

func StatementUri(statement string) string {
	hash := sha256.New()
	hash.Write([]byte(statement))
	return fmt.Sprintf("hash://sha256/%x", hash.Sum(nil))
}

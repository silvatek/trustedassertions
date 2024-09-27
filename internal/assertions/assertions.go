package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
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

type Referenceable interface {
	Uri() string
	Type() string
	Content() string
}

//============================================//

type Statement struct {
	uri     string
	content string
}

func NewStatement(content string) Statement {
	return Statement{content: content}
}

func (s *Statement) Uri() string {
	if s.uri == "" {
		hash := sha256.New()
		hash.Write([]byte(s.content))
		return fmt.Sprintf("hash://sha256/%x", hash.Sum(nil))
	}
	return s.uri
}

func (s *Statement) Type() string {
	return "Statement"
}

func (s *Statement) Content() string {
	return s.content
}

//============================================//

type Entity struct {
	SerialNum   big.Int
	CommonName  string `json:"name"`
	Certificate string `json:"cert"`
}

func NewEntity(commonName string) Entity {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)
	return Entity{CommonName: commonName, SerialNum: *serNum}
}

func (e *Entity) Uri() string {
	if !e.HasSerialNum() {
		e.AssignSerialNum()
	}
	return fmt.Sprintf("cert://x509/%s", e.SerialNum.String())
}

func (e *Entity) AssignSerialNum() {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)

	e.SerialNum = *serNum
}

func (e *Entity) HasSerialNum() bool {
	return e.SerialNum.Int64() != 0
}

func (e *Entity) Type() string {
	return "Entity"
}

func (e *Entity) Content() string {
	return e.Certificate
}

func (e *Entity) MakeCertificate() {
	if !e.HasSerialNum() {
		e.AssignSerialNum()
	}

	template := x509.Certificate{
		SerialNumber:          &e.SerialNum,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{Organization: []string{e.CommonName}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 2),
		BasicConstraintsValid: true,
	}
	cert, _ := x509.CreateCertificate(rand.Reader, &template, &template, &PublicKey, PrivateKey)
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert}
	e.Certificate = string(pem.EncodeToMemory(&b))
}

func ParseCertificate(content string) Entity {
	entity := NewEntity("{unknown}")

	entity.Certificate = content

	return entity
}

//============================================//

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

// func ParseEntityJwt(token string) (Entity, error) {
// 	entity := Entity{
// 		RegisteredClaims: &jwt.RegisteredClaims{},
// 	}
// 	_, err := jwt.ParseWithClaims(token, &entity, verificationKey)
// 	return entity, err
// }

func MakeEntityCertificate(entity *Entity) {
	// max := new(big.Int)
	// max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	// serNum, _ := rand.Int(rand.Reader, max)
	if !entity.HasSerialNum() {
		entity.AssignSerialNum()
	}

	template := x509.Certificate{
		SerialNumber:          &entity.SerialNum,
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

//============================================//

func InitKeyPair() {
	osKey := os.Getenv("PRV_KEY")
	if osKey == "" {
		log.Println("Generating new key pair")
		PrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		log.Println("Loading private key from environment")
		bytes, _ := base64.StdEncoding.DecodeString(osKey)
		PrivateKey, _ = x509.ParsePKCS1PrivateKey(bytes)
	}
	PublicKey = PrivateKey.PublicKey
}

func Base64Private() string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(PrivateKey))
}

//============================================//

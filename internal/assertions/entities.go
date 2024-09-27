package assertions

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"time"
)

type Entity struct {
	SerialNum   big.Int
	CommonName  string `json:"name"`
	Certificate string `json:"cert"`
}

func NewEntity(commonName string, serialNum big.Int) Entity {
	return Entity{CommonName: commonName, SerialNum: serialNum}
}

func randomSerialNum() big.Int {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)

	return *serNum
}

func (e *Entity) Uri() string {
	if !e.HasSerialNum() {
		e.AssignSerialNum()
	}
	return fmt.Sprintf("cert://x509/%s", e.SerialNum.String())
}

func (e *Entity) AssignSerialNum() {
	e.SerialNum = randomSerialNum()
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
		Subject:               pkix.Name{CommonName: e.CommonName},
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
	entity := NewEntity("{unknown}", *big.NewInt(0))

	entity.Certificate = content

	p, _ := pem.Decode([]byte(content))

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		log.Print(err)
	} else {
		entity.SerialNum = *cert.SerialNumber
		entity.CommonName = cert.Subject.CommonName
		log.Printf("Entity serial number: %d", entity.SerialNum.Int64())
		log.Printf("Entity name: %s", entity.CommonName)
	}

	return entity
}

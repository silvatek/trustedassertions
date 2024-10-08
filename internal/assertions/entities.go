package assertions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	log "silvatek.uk/trustedassertions/internal/logging"
)

type Entity struct {
	SerialNum   big.Int
	CommonName  string         `json:"name"`
	Certificate string         `json:"cert"`
	uri         HashUri        `json:"-"`
	Issued      time.Time      `json:"-"`
	PublicKey   *rsa.PublicKey `json:"-"`
}

func NewEntity(commonName string, serialNum big.Int) Entity {
	return Entity{CommonName: commonName, SerialNum: serialNum, Issued: time.Now()}
}

func randomSerialNum() big.Int {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))

	serNum, _ := rand.Int(rand.Reader, max)

	return *serNum
}

func (e *Entity) Uri() HashUri {
	if e.uri.String() == "" {
		if e.Certificate == "" {
			log.Errorf("Cannot make URI without certificate")
			return EMPTY_URI
		}
		hash := sha256.New()
		hash.Write([]byte(e.Certificate))
		e.uri = MakeUriB(hash.Sum(nil), "entity")
	}
	return e.uri
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

func (e *Entity) Summary() string {
	return e.CommonName
}

func (e *Entity) MakeCertificate(privateKey *rsa.PrivateKey) {
	if !e.HasSerialNum() {
		e.AssignSerialNum()
	}

	template := x509.Certificate{
		SerialNumber:          &e.SerialNum,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{CommonName: e.CommonName},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             e.Issued,
		NotAfter:              e.Issued.Add(time.Hour * 24 * 365 * 2),
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Errorf("Error creating entity certificate: %v", err)
		return
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert}
	e.Certificate = string(pem.EncodeToMemory(&b))
}

func ParseCertificate(content string) Entity {
	entity := NewEntity("{unknown}", *big.NewInt(0))

	entity.Certificate = content

	p, _ := pem.Decode([]byte(content))

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		log.Errorf("Error parsing X509 certificate: %v", err)
	} else {
		entity.SerialNum = *cert.SerialNumber
		entity.CommonName = cert.Subject.CommonName
		entity.PublicKey = cert.PublicKey.(*rsa.PublicKey)
		log.Debugf("Entity serial number: %d", entity.SerialNum.Int64())
		log.Debugf("Entity name: %s", entity.CommonName)
	}

	return entity
}

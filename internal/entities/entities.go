package entities

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	log "silvatek.uk/trustedassertions/internal/logging"
	refs "silvatek.uk/trustedassertions/internal/references"
)

type Entity struct {
	SerialNum   big.Int
	CommonName  string         `json:"name"`
	Certificate string         `json:"cert"`
	uri         refs.HashUri   `json:"-"`
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

func (e *Entity) Uri() refs.HashUri {
	if e.uri.String() == "" {
		if e.Certificate == "" {
			log.Errorf("Cannot make URI without certificate")
			return refs.ERROR_URI
		}
		e.uri = refs.UriFor(e)
		// hash := sha256.New()
		// hash.Write([]byte(e.Certificate))
		// e.uri = MakeUriB(hash.Sum(nil), "entity")
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

func (e *Entity) TextContent() string {
	return e.CommonName
}

func (e *Entity) References() []refs.HashUri {
	return []refs.HashUri{}
}

func (e *Entity) MakeCertificate(privateKey *rsa.PrivateKey) {
	if !e.HasSerialNum() {
		e.AssignSerialNum()
	}
	e.PublicKey = &privateKey.PublicKey

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

func (e *Entity) ParseContent(content string) error {
	e.Certificate = content

	p, _ := pem.Decode([]byte(content))

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return err
	} else {
		e.SerialNum = *cert.SerialNumber
		e.CommonName = cert.Subject.CommonName
		e.PublicKey = cert.PublicKey.(*rsa.PublicKey)
		return nil
	}
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
	}

	return entity
}

func PrivateKeyToString(prvKey *rsa.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(prvKey))
}

func PrivateKeyFromString(base64encoded string) *rsa.PrivateKey {
	bytes, _ := base64.StdEncoding.DecodeString(base64encoded)
	privateKey, err := x509.ParsePKCS1PrivateKey(bytes)
	if err != nil {
		log.Errorf("Error decoding key")
	}
	return privateKey
}

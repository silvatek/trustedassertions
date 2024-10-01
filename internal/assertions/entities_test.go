package assertions

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestEntityUri(t *testing.T) {
	entity := NewEntity("Tester", *big.NewInt(0))
	entity.AssignSerialNum()
	uri := entity.Uri()
	if uri == "" {
		t.Error("Empty entity URI")
	}
	if !strings.HasPrefix(uri, "hash://sha256/") {
		t.Errorf("Entity URI does not have correct prefix: %s", uri)
	}
}

func TestCreateCertificate(t *testing.T) {
	InitKeyPair("")

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
	InitKeyPair("")

	entity := &Entity{
		CommonName: "John Smith",
	}

	entity.MakeCertificate()

	if entity.Certificate == "" {
		t.Error(entity)
	}
}

func TestEntityUriRoundTrip(t *testing.T) {
	InitKeyPair("")

	e1 := NewEntity("Anentity", *big.NewInt(123456))
	e1.MakeCertificate()
	c1 := e1.Certificate
	u1 := e1.Uri()

	e2 := ParseCertificate(c1)
	u2 := e2.Uri()

	if u1 != u2 {
		t.Errorf("Round trip URI mismatch: %s != %s", u1, u2)
	}
}

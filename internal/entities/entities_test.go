package entities

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestEntityUri(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := NewEntity("Tester", *big.NewInt(0))
	entity.AssignSerialNum()
	entity.MakeCertificate(privateKey)
	uri := entity.Uri()
	if uri.IsEmpty() {
		t.Error("Empty entity URI")
	}
	if !strings.HasPrefix(uri.String(), "hash://sha256/") {
		t.Errorf("Entity URI does not have correct prefix: %s", uri)
	}
	if entity.Type() != "Entity" {
		t.Errorf("Unexpected entity type: %s", entity.Type())
	}
	if entity.Summary() != "Tester" {
		t.Errorf("Unexpected entity summary: %s", entity.Summary())
	}
	if entity.Content() == "" {
		t.Error("Entity content is empty after making certificate")
	}
}

func TestCreateCertificate(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := privateKey.PublicKey

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
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &publicKey, privateKey)
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
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	entity := &Entity{
		CommonName: "John Smith",
	}

	entity.MakeCertificate(privateKey)

	if entity.Certificate == "" {
		t.Error(entity)
	}
}

func TestEntityUriRoundTrip(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	e1 := NewEntity("Anentity", *big.NewInt(123456))
	e1.MakeCertificate(privateKey)
	c1 := e1.Certificate
	u1 := e1.Uri()

	e2 := ParseCertificate(c1)
	u2 := e2.Uri()

	if u1 != u2 {
		t.Errorf("Round trip URI mismatch: %s != %s", u1, u2)
	}
}

func TestPrivateKeyToFromString(t *testing.T) {
	key1, _ := rsa.GenerateKey(rand.Reader, 2048)

	keyStr := PrivateKeyToString(key1)

	key2 := PrivateKeyFromString(keyStr)

	if key2.N.Cmp(key1.N) != 0 {
		t.Errorf("Key does not match after round trip: %v != %v", key2.N, key1.N)
	}
}

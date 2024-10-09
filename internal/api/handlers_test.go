package api

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestStatmentApi(t *testing.T) {
	router := mux.NewRouter()
	AddHandlers(router)

	statement := assertions.NewStatement("test")

	datastore.InitInMemoryDataStore()
	datastore.ActiveDataStore.Store(&statement)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", statement.Uri().ApiPath(), nil)

	router.ServeHTTP(w, r)

	response := w.Body.String()
	if response != "test" {
		t.Errorf("Unexpected response body: %s", response)
	}
}

func TestEntityApi(t *testing.T) {
	router := mux.NewRouter()
	AddHandlers(router)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := assertions.NewEntity("Test", *big.NewInt(1234))
	entity.MakeCertificate(privateKey)

	datastore.InitInMemoryDataStore()
	datastore.ActiveDataStore.Store(&entity)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", entity.Uri().ApiPath(), nil)

	router.ServeHTTP(w, r)

	response := w.Body.String()
	if response[0:10] != "-----BEGIN" {
		t.Errorf("Unexpected response body: %s", response)
	}
}

func TestAssertionApi(t *testing.T) {
	router := mux.NewRouter()
	AddHandlers(router)

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	entity := assertions.NewEntity("Test", *big.NewInt(1234))
	entity.MakeCertificate(privateKey)

	statement := assertions.NewStatement("test")

	datastore.InitInMemoryDataStore()
	assertions.ActiveEntityFetcher = datastore.ActiveDataStore

	datastore.ActiveDataStore.Store(&entity)
	datastore.ActiveDataStore.Store(&statement)

	assertion := assertions.NewAssertion("IsTrue")
	assertion.Issuer = entity.Uri().String()
	assertion.Subject = statement.Uri().String()
	assertion.MakeJwt(privateKey)
	datastore.ActiveDataStore.Store(&assertion)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", assertion.Uri().ApiPath(), nil)

	router.ServeHTTP(w, r)

	response := w.Body.String()
	if response[0:4] != "eyJh" {
		t.Errorf("Unexpected response body: %s", response)
	}
}

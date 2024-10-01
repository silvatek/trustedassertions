package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/api/v1/statements/{key}", StatementApiHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityApiHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionApiHandler)
}

func StatementApiHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	log.Debugf("Statement key: %s", key)

	statement, err := datastore.ActiveDataStore.FetchStatement("hash://sha256/" + key)
	if err != nil {
		log.Errorf("Error fetching statement: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(statement.Content()))
}

func EntityApiHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	entity, _ := datastore.ActiveDataStore.FetchEntity("hash://sha256/" + key)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write([]byte(entity.Certificate))
}

func AssertionApiHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	assertion, err := datastore.ActiveDataStore.FetchAssertion("hash://sha256/" + key)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(assertion.Content()))
}
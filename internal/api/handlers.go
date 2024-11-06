package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/api/v1/statements/{key}", StatementApiHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityApiHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionApiHandler)

	r.HandleFunc("/api/v1/reindex", ReindexApiHandler)
}

func StatementApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	log.Debugf("Statement key: %s", key)

	statement, err := datastore.ActiveDataStore.FetchStatement(ctx, assertions.MakeUri(key, "statement"))
	if err != nil {
		log.Errorf("Error fetching statement: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(statement.Content()))
}

func EntityApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	entity, _ := datastore.ActiveDataStore.FetchEntity(ctx, assertions.MakeUri(key, "entity"))

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write([]byte(entity.Certificate))
}

func AssertionApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	assertion, err := datastore.ActiveDataStore.FetchAssertion(ctx, assertions.MakeUri(key, "assertion"))

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

func ReindexApiHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Reindexing..."))

	datastore.ActiveDataStore.Reindex()

	w.Write([]byte("Done"))
}

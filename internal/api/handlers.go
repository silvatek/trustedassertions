package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/references"
)

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/api/v1/statements/{key}", StatementApiHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityApiHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionApiHandler)

	r.HandleFunc("/api/v1/passkey/beginregistration", PassKeyRegStartHandler)
	r.HandleFunc("/api/v1/passkey/completeregistration", PassKeyRegFinishHandler)

	//r.HandleFunc("/api/v1/reindex", ReindexApiHandler)
}

func StatementApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	log.Debugf("Statement key: %s", key)

	statement, err := datastore.ActiveDataStore.FetchStatement(ctx, references.MakeUri(key, "statement"))
	if err != nil {
		log.Errorf("Error fetching statement: %v", err)
	}

	setHeaders(w, r, http.StatusOK, "text/plain")
	w.Write([]byte(statement.Content()))
}

func EntityApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	entity, _ := datastore.ActiveDataStore.FetchEntity(ctx, references.MakeUri(key, "entity"))

	setHeaders(w, r, http.StatusOK, "text/plain")
	w.Write([]byte(entity.Certificate))
}

func AssertionApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["key"]
	assertion, err := datastore.ActiveDataStore.FetchAssertion(ctx, references.MakeUri(key, "assertion"))

	if err != nil {
		setHeaders(w, r, http.StatusInternalServerError, "text/plain")
		w.Write([]byte(err.Error()))
		return
	}

	setHeaders(w, r, http.StatusOK, "text/plain")
	w.Write([]byte(assertion.Content()))
}

func ReindexApiHandler(w http.ResponseWriter, r *http.Request) {
	setHeaders(w, r, http.StatusOK, "text/plain")
	w.Write([]byte("Reindexing..."))

	datastore.ActiveDataStore.Reindex()

	w.Write([]byte("Done"))
}

func setHeaders(w http.ResponseWriter, r *http.Request, httpStatus int, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(httpStatus)
}

func PassKeyRegStartHandler(w http.ResponseWriter, r *http.Request) {
	reg := auth.BeginPasskeyRegistration()
	setHeaders(w, r, http.StatusOK, "application/json")
	json.NewEncoder(w).Encode(reg.Options)
}

func PassKeyRegFinishHandler(w http.ResponseWriter, r *http.Request) {
	auth.FinishPasskeyRegistration(r)
	setHeaders(w, r, http.StatusOK, "application/json")
	json.NewEncoder(w).Encode("OK")
}

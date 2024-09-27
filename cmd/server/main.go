package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"

	"github.com/gorilla/mux"
)

// var entityKey string
var statementUri string
var entityUri string

func main() {
	log.Print("Starting TrustedAssertions server...")

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/api/v1/statements/{key}", StatementHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityHandler)

	assertions.InitKeyPair()
	datastore.InitInMemoryDataStore()

	log.Print(assertions.Base64Private())

	statement := assertions.NewStatement("The world is flat")
	statementUri = statement.Uri()
	datastore.DataStore.Store(&statement)

	entity := assertions.NewEntity("Fred Bloggs")
	entityUri = entity.Uri()
	entity.MakeCertificate()
	datastore.DataStore.Store(&entity)

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddress(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Welcome to TrustedAssertions\n")
}

func StatementHandler(w http.ResponseWriter, r *http.Request) {
	statement := datastore.DataStore.FetchStatement(statementUri)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte([]byte(statement.Content())))
}

func EntityHandler(w http.ResponseWriter, r *http.Request) {
	entity := datastore.DataStore.FetchEntity(entityUri)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write([]byte(entity.Certificate))
}

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"

	"github.com/gorilla/mux"
)

var entityKey string

func main() {
	log.Print("Starting TrustedAssertions server...")

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/api/v1/entity", EntityHandler)

	assertions.InitKeyPair()
	datastore.InitInMemoryDataStore()

	entity := &assertions.Entity{
		CommonName: "John Smith",
		PublicKey:  assertions.PublicKey.N.String(),
	}
	entityKey = datastore.DataStore.StoreClaims(entity)

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

func EntityHandler(w http.ResponseWriter, r *http.Request) {
	entity, _ := datastore.DataStore.FetchEntity(entityKey)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	e.Encode(entity)
}

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

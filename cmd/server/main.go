package main

import (
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"

	"github.com/gorilla/mux"
)

func main() {
	log.Print("Starting TrustedAssertions server...")

	r := setupHandlers()

	assertions.InitKeyPair()
	initDataStore()

	if datastore.ActiveDataStore.Name() == "InMemoryDataStore" {
		setupTestData()
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddress(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func setupHandlers() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/api/v1/statements/{key}", StatementHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionHandler)
	r.HandleFunc("/api/v1/initdb", InitDbHandler)

	staticDir := http.Dir("./web/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	return r
}

var entityUri string

func initDataStore() {
	if os.Getenv("FIRESTORE_DB_NAME") != "" {
		datastore.InitFireStore()
	} else {
		datastore.InitInMemoryDataStore()
	}
}

func setupTestData() {
	log.Printf("Loading test data into %s", datastore.ActiveDataStore.Name())

	statement := assertions.NewStatement("The world is flat")
	datastore.ActiveDataStore.Store(&statement)
	log.Printf("Statement URI: %s", statement.Uri())

	entity := assertions.NewEntity("Fred Bloggs", *big.NewInt(2337203685477580792))
	entity.MakeCertificate()
	datastore.ActiveDataStore.Store(&entity)
	entityUri = entity.Uri()
	log.Printf("Entity URI: %s", entityUri)

	assertion := assertions.NewAssertion("IsFalse")
	assertion.Subject = statement.Uri()
	assertion.SetAssertingEntity(entity)
	assertion.Confidence = 0.9
	datastore.ActiveDataStore.Store(&assertion)
	log.Printf("Assertion URI: %s", assertion.Uri())
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	dir := "./web"
	templateName := "index"
	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+templateName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Printf("Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	data := struct {
		EntityHash string
	}{
		EntityHash: strings.TrimPrefix(entityUri, "hash://sha256/"),
	}

	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Printf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func InitDbHandler(w http.ResponseWriter, r *http.Request) {
	setupTestData()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Data store initialised"))
}

func StatementHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	log.Printf("Statement key: %s", key)

	statement := datastore.ActiveDataStore.FetchStatement("hash://sha256/" + key)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(statement.Content()))
}

func EntityHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	entity := datastore.ActiveDataStore.FetchEntity("hash://sha256/" + key)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write([]byte(entity.Certificate))
}

func AssertionHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	assertion := datastore.ActiveDataStore.FetchAssertion("hash://sha256/" + key)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/jwt")
	w.Write([]byte(assertion.Content()))
}

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

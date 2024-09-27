package main

import (
	"html/template"
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
var assertionUri string

func main() {
	log.Print("Starting TrustedAssertions server...")

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/api/v1/statements/{key}", StatementHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionHandler)

	assertions.InitKeyPair()
	datastore.InitInMemoryDataStore()

	//log.Print(assertions.Base64Private())

	setupTestData()

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddress(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func setupTestData() {
	statement := assertions.NewStatement("The world is flat")
	statementUri = statement.Uri()
	datastore.DataStore.Store(&statement)

	entity := assertions.NewEntity("Fred Bloggs")
	entityUri = entity.Uri()
	entity.MakeCertificate()
	datastore.DataStore.Store(&entity)

	assertion := assertions.NewAssertion("simple")
	assertionUri = assertion.Uri()
	datastore.DataStore.Store(&assertion)
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

	data := "test"

	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Printf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}

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

func AssertionHandler(w http.ResponseWriter, r *http.Request) {
	assertion := datastore.DataStore.FetchAssertion(assertionUri)

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

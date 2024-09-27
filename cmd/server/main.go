package main

import (
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"

	"github.com/gorilla/mux"
)

func main() {
	log.Print("Starting TrustedAssertions server...")

	log.Printf("FireStore DB name: %s", os.Getenv("FIRESTORE_DB_NAME"))

	r := setupHandlers()

	assertions.InitKeyPair()
	initDataStore()

	setupTestData()

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

	staticDir := http.Dir("./web/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	return r
}

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

	entity := assertions.NewEntity("Fred Bloggs", *big.NewInt(2337203685477580792))
	entity.MakeCertificate()
	datastore.ActiveDataStore.Store(&entity)

	assertion := assertions.NewAssertion("simple")
	datastore.ActiveDataStore.Store(&assertion)
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
	key := mux.Vars(r)["key"]
	log.Printf("Statement key: %s", key)

	statement := datastore.ActiveDataStore.FetchStatement("hash://sha256/" + key)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte([]byte(statement.Content())))
}

func EntityHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	entityUri := "cert://x509/" + key
	log.Printf("Entity URI: %s", entityUri)
	entity := datastore.ActiveDataStore.FetchEntity(entityUri)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write([]byte(entity.Certificate))
}

func AssertionHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	assertion := datastore.ActiveDataStore.FetchAssertion("sig://jwt/" + key)

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

package main

import (
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"

	"github.com/gorilla/mux"
)

func main() {
	initLogging()
	log.Print("Starting TrustedAssertions server...")

	assertions.InitKeyPair()
	initDataStore()

	r := setupHandlers()

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
	r.HandleFunc("/", HomeWebHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)
	r.HandleFunc("/api/v1/statements/{key}", StatementApiHandler)
	r.HandleFunc("/api/v1/entities/{key}", EntityApiHandler)
	r.HandleFunc("/api/v1/assertions/{key}", AssertionApiHandler)
	r.HandleFunc("/api/v1/initdb", InitDbApiHandler)

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
		setupTestData()
	}
}

func initLogging() {
	log.StructureLogs = (os.Getenv("FIRESTORE_DB_NAME") != "")
}

func setupTestData() {
	log.Infof("Loading test data into %s", datastore.ActiveDataStore.Name())

	files, err := os.ReadDir("./testdata/")
	if err != nil {
		log.Errorf("Error reading directory: %v", err)
	}

	for _, file := range files {
		uri := "hash://sha256/" + strings.TrimSuffix(file.Name(), ".txt")

		content, err := os.ReadFile("./testdata/" + file.Name())
		if err != nil {
			log.Errorf("Error reading file %s, %v", file.Name(), err)
		} else {
			datastore.ActiveDataStore.StoreRaw(uri, string(content))
		}
	}

	log.Info("Test data load complete.")
}

func RenderWebPage(pageName string, data interface{}, w http.ResponseWriter) {
	dir := "./web"

	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+pageName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func HomeWebHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		EntityHash string
	}{
		EntityHash: strings.TrimPrefix(entityUri, "hash://sha256/"),
	}

	RenderWebPage("index", data, w)
}

func NewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		RenderWebPage("newstatementform", "", w)
	} else if r.Method == "POST" {
		r.ParseForm()
		content := r.Form.Get("statement")
		log.Debugf("Web post of new statement: %s", content)

		// Create and save the statement
		statement := assertions.NewStatement(content)
		datastore.ActiveDataStore.Store(&statement)

		// Fetch the default entity
		entity, _ := datastore.ActiveDataStore.FetchEntity("hash://sha256/c6355ef5dfbc9da513ac4d683729ee24209aa1f9a8afe66bb4aa60217439183f")

		// Create and save an assertion by the default entity that the statement is probably true
		assertion := assertions.NewAssertion("IsTrue")
		assertion.Subject = statement.Uri()
		assertion.Confidence = 0.8
		assertion.SetAssertingEntity(entity)
		datastore.ActiveDataStore.Store(&assertion)

		// Redirect the user to the assertion
		hash := strings.TrimPrefix(assertion.Uri(), "hash://sha256/")
		http.Redirect(w, r, "/api/v1/assertions/"+hash, http.StatusSeeOther)
	}
}

func InitDbApiHandler(w http.ResponseWriter, r *http.Request) {
	setupTestData()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Data store initialised"))
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

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

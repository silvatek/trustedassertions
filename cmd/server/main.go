package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/api"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/web"

	"github.com/gorilla/mux"
)

var testDataDir string
var defaultEntityUri string
var defaultEntityKey string

func main() {
	initLogging()
	log.Print("Starting TrustedAssertions server...")

	testDataDir = "./testdata"
	initDataStore()

	web.TemplateDir = "./web"
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

	api.AddHandlers(r)

	web.AddHandlers(r)

	r.HandleFunc("/api/v1/initdb", InitDbApiHandler)

	return r
}

func initDataStore() {
	if os.Getenv("FIRESTORE_DB_NAME") != "" {
		datastore.InitFireStore()
	} else {
		datastore.InitInMemoryDataStore()
	}

	if defaultEntityUri == "" {
		defaultEntityUri = os.Getenv("DEFAULT_ENTITY")
		web.DefaultEntityUri = assertions.UriFromString(defaultEntityUri)
	}

	if defaultEntityKey == "" {
		defaultEntityKey = os.Getenv("PRV_KEY")
	}

	assertions.ActiveEntityFetcher = datastore.ActiveDataStore

	if datastore.ActiveDataStore.AutoInit() {
		setupTestData()
	}
}

func initLogging() {
	log.StructureLogs = (os.Getenv("GCLOUD_PROJECT") != "")
}

func setupTestData() {
	log.Infof("Loading test data into %s", datastore.ActiveDataStore.Name())

	loadTestData(testDataDir+"/entities", "Entity")

	if defaultEntityUri != "" {
		uri := assertions.UriFromString(defaultEntityUri)
		datastore.ActiveDataStore.StoreKey(uri, defaultEntityKey)
	}

	loadTestData(testDataDir+"/statements", "Statement")
	loadTestData(testDataDir+"/assertions", "Assertion")

	initialUser := auth.User{Id: os.Getenv("INITIAL_USER")}
	initialUser.HashPassword(os.Getenv("INITIAL_PW"))
	initialUser.AddKeyRef(defaultEntityUri, "Default")
	datastore.ActiveDataStore.StoreUser(initialUser)

	log.Info("Test data load complete.")
}

func loadTestData(dirName string, dataType string) {
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Errorf("Error reading directory: %v", err)
	}

	for _, file := range files {
		hash := strings.TrimSuffix(file.Name(), ".txt")
		uri := assertions.MakeUri(hash, dataType)

		content, err := os.ReadFile(dirName + "/" + file.Name())
		if err != nil {
			log.Errorf("Error reading file %s, %v", file.Name(), err)
		} else {
			datastore.ActiveDataStore.StoreRaw(uri, string(content))
		}

		if dataType == "assertion" {
			addAssertionReferences(string(content))
		}
	}
}

func addAssertionReferences(content string) {
	a, _ := assertions.ParseAssertionJwt(content)
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Subject), "Assertion.Subject:Statement")
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Issuer), "Assertion.Issuer:Entity")
}

func InitDbApiHandler(w http.ResponseWriter, r *http.Request) {
	setupTestData()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Data store initialised"))
}

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

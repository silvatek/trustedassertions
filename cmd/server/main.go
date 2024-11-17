package main

import (
	"net/http"
	"os"
	"time"

	"silvatek.uk/trustedassertions/internal/api"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/testdata"
	"silvatek.uk/trustedassertions/internal/web"

	"github.com/gorilla/csrf"
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

	CSRF := csrf.Protect(
		[]byte(os.Getenv("CSRF_KEY")),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.FieldName("authenticity_token"),
		csrf.Path("/"),
		csrf.CookieName("authenticity_token"),
	)

	srv := &http.Server{
		Handler:      CSRF(r),
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
		web.DefaultEntityUri = UriFromString(defaultEntityUri)
	}

	if defaultEntityKey == "" {
		defaultEntityKey = os.Getenv("PRV_KEY")
	}

	assertions.PublicKeyResolver = datastore.ActiveDataStore

	if datastore.ActiveDataStore.AutoInit() {
		testdata.SetupTestData(testDataDir, defaultEntityUri, defaultEntityKey)
	}
}

func initLogging() {
	log.StructureLogs = (os.Getenv("GCLOUD_PROJECT") != "")
}

func InitDbApiHandler(w http.ResponseWriter, r *http.Request) {
	testdata.SetupTestData(testDataDir, defaultEntityUri, defaultEntityKey)
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

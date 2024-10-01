package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/api"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/web"

	"github.com/gorilla/mux"
)

func main() {
	initLogging()
	log.Print("Starting TrustedAssertions server...")

	assertions.InitKeyPair(os.Getenv("PRV_KEY"))

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
		setupTestData()
	}
}

func initLogging() {
	log.StructureLogs = (os.Getenv("GCLOUD_PROJECT") != "")
}

func setupTestData() {
	log.Infof("Loading test data into %s", datastore.ActiveDataStore.Name())

	files, err := os.ReadDir("./testdata/")
	if err != nil {
		log.Errorf("Error reading directory: %v", err)
	}

	for _, file := range files {
		hash := strings.TrimSuffix(file.Name(), ".txt")
		uri := "hash://sha256/" + hash

		content, err := os.ReadFile("./testdata/" + file.Name())
		if err != nil {
			log.Errorf("Error reading file %s, %v", file.Name(), err)
		} else {
			datastore.ActiveDataStore.StoreRaw(uri, string(content))
		}
	}

	datastore.ActiveDataStore.StoreKey("hash://sha256/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db", assertions.Base64Private())

	log.Info("Test data load complete.")
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

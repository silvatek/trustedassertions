package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"silvatek.uk/trustedassertions/internal/assertions"

	"github.com/gorilla/mux"
)

func main() {
	log.Print("Starting TrustedAssertions server...")
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)

	assertions.IntiKeyPair()

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
	fmt.Fprintf(w, "Welcome to TrustedAssertions")
}

func listenAddress() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return ":" + envPort
	} else {
		return "127.0.0.1:8080"
	}
}

package testdata

import (
	"context"
	"os"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/references"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

func SetupTestData(testDataDir string, defaultEntityUri string, defaultEntityKey string) {
	log.Infof("Loading test data into %s", datastore.ActiveDataStore.Name())

	loadTestData(testDataDir+"/entities", "Entity", "txt", false)

	if defaultEntityUri != "" {
		uri := UriFromString(defaultEntityUri)
		datastore.ActiveDataStore.StoreKey(uri, defaultEntityKey)
	}

	loadTestData(testDataDir+"/statements", "Statement", "txt", false)
	loadTestData(testDataDir+"/assertions", "Assertion", "txt", false)
	loadTestData(testDataDir+"/documents", "Document", "xml", true)

	initialUser := auth.User{Id: os.Getenv("INITIAL_USER")}
	initialUser.HashPassword(os.Getenv("INITIAL_PW"))
	initialUser.AddKeyRef(defaultEntityUri, "Default")
	datastore.ActiveDataStore.StoreUser(initialUser)

	log.Info("Test data load complete.")
}

func loadTestData(dirName string, dataType string, extension string, calcHash bool) {
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Errorf("Error reading directory: %v", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(dirName + "/" + file.Name())
		if err != nil {
			log.Errorf("Error reading file %s, %v", file.Name(), err)
			continue
		}

		content = statements.NormalizeNewlines(content)

		item := assertions.NewReferenceable(dataType)
		item.ParseContent(string(content))

		datastore.ActiveDataStore.Store(context.TODO(), item)

		if strings.ToLower(dataType) == "assertion" {
			addAssertionReferences(string(content))
		}
	}
}

func addAssertionReferences(content string) {
	assertion, _ := assertions.ParseAssertionJwt(content)
	datastore.CreateReferenceWithSummary(context.Background(), assertion.Uri(), references.UriFromString(assertion.Subject))
	datastore.CreateReferenceWithSummary(context.Background(), assertion.Uri(), references.UriFromString(assertion.Issuer))
}

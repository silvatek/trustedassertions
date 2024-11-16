package testdata

import (
	"bytes"
	"context"
	"os"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
	. "silvatek.uk/trustedassertions/internal/references"
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

		content = NormalizeNewlines(content)

		item := assertions.NewReferenceable(dataType)
		item.ParseContent(string(content))

		datastore.ActiveDataStore.Store(context.TODO(), item)

		if strings.ToLower(dataType) == "assertion" {
			addAssertionReferences(string(content))
		}
	}
}

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}

func addAssertionReferences(content string) {
	a, _ := assertions.ParseAssertionJwt(content)
	datastore.StoreReference(a.Uri(), UriFromString(a.Subject), "Assertion.Subject:Statement")
	datastore.StoreReference(a.Uri(), UriFromString(a.Issuer), "Assertion.Issuer:Entity")
}

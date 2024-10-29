package testdata

import (
	"bytes"
	"crypto/sha256"
	"os"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/docs"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func SetupTestData(testDataDir string, defaultEntityUri string, defaultEntityKey string) {
	log.Infof("Loading test data into %s", datastore.ActiveDataStore.Name())

	loadTestData(testDataDir+"/entities", "Entity", "txt", false)

	if defaultEntityUri != "" {
		uri := assertions.UriFromString(defaultEntityUri)
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

		var uri assertions.HashUri
		if calcHash {
			content := NormalizeNewlines(content)
			hash := sha256.New()
			hash.Write(content)
			uri = assertions.MakeUriB(hash.Sum(nil), dataType)
			if dataType == "Document" {
				docs.DefaultDocumentUri = uri
			}
		} else {
			hash := strings.TrimSuffix(file.Name(), ".txt")
			uri = assertions.MakeUri(hash, dataType)
		}

		datastore.ActiveDataStore.StoreRaw(uri, string(content))

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
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Subject), "Assertion.Subject:Statement")
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Issuer), "Assertion.Issuer:Entity")
}

package testdata

import (
	"context"
	"os"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/logging"
	ref "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var log = logging.GetLogger("testdataloader")

func SetupTestData(ctx context.Context, testDataDir string, defaultEntityUri string, defaultEntityKey string) {
	log.InfofX(ctx, "Loading test data into %s", datastore.ActiveDataStore.Name())

	loadTestData(ctx, testDataDir+"/entities", "Entity", "txt", false)

	if defaultEntityUri != "" {
		uri := ref.UriFromString(defaultEntityUri)
		datastore.ActiveDataStore.StoreKey(uri, defaultEntityKey)
	}

	loadTestData(ctx, testDataDir+"/statements", "Statement", "txt", false)
	loadTestData(ctx, testDataDir+"/assertions", "Assertion", "txt", false)
	loadTestData(ctx, testDataDir+"/documents", "Document", "xml", true)

	initialUser := auth.User{Id: os.Getenv("INITIAL_USER")}
	initialUser.HashPassword(os.Getenv("INITIAL_PW"))
	initialUser.AddKeyRef(defaultEntityUri, "Default")
	datastore.ActiveDataStore.StoreUser(ctx, initialUser)

	datastore.ActiveDataStore.StoreRegistration(ctx, auth.Registration{Code: "TESTCODE-1001", Status: "Pending"})

	log.InfofX(ctx, "Test data load complete.")
}

func loadTestData(ctx context.Context, dirName string, dataType string, extension string, calcHash bool) {
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.ErrorfX(ctx, "Error reading directory: %v", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(dirName + "/" + file.Name())
		if err != nil {
			log.ErrorfX(ctx, "Error reading file %s, %v", file.Name(), err)
			continue
		}

		content = statements.NormalizeNewlines(content)

		item := assertions.NewReferenceable(dataType)
		item.ParseContent(string(content))

		datastore.ActiveDataStore.Store(ctx, item)

		if strings.ToLower(dataType) == "assertion" {
			addAssertionReferences(ctx, string(content))
		}
	}
}

func addAssertionReferences(ctx context.Context, content string) {
	assertion, _ := assertions.ParseAssertionJwt(content)
	datastore.CreateReferenceWithSummary(ctx, assertion.Uri(), ref.UriFromString(assertion.Subject))
	datastore.CreateReferenceWithSummary(ctx, assertion.Uri(), ref.UriFromString(assertion.Issuer))
}

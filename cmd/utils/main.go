package main

import (
	"math/big"
	"os"

	"silvatek.uk/trustedassertions/internal/assertions"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func main() {

	log.StructureLogs = (os.Getenv("GCLOUD_PROJECT") != "")
	log.Info("TrustedAssertions utils")

	assertions.InitKeyPair(os.Getenv("PRV_KEY"))
	key := assertions.Base64Private()
	log.Infof("Key: %s", key[len(key)-8:])

	entity := assertions.NewEntity("Mr Tester", *big.NewInt(8376446832743937489))
	entity.MakeCertificate()

	log.Infof("Certificate URI: %s", entity.Uri())

	hash := assertions.UriHash(entity.Uri())

	dirName := "./testdata"

	err := os.WriteFile(dirName+"/"+hash+".txt", []byte(entity.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}

	statement := assertions.NewStatement("The universe exists")
	log.Infof("Statement URI: %s", statement.Uri())
	hash = assertions.UriHash(statement.Uri())

	err = os.WriteFile(dirName+"/"+hash+".txt", []byte(statement.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}

	assertion := assertions.NewAssertion("IsTrue")
	assertion.Subject = statement.Uri()
	assertion.SetAssertingEntity(entity)
	log.Infof("Assertion URI: %s", assertion.Uri())
	hash = assertions.UriHash(assertion.Uri())

	err = os.WriteFile(dirName+"/"+hash+".txt", []byte(assertion.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}
}

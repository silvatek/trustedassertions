package main

import (
	"math/big"
	"os"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/entities"
	log "silvatek.uk/trustedassertions/internal/logging"
	"silvatek.uk/trustedassertions/internal/statements"
)

func main() {

	log.StructureLogs = (os.Getenv("GCLOUD_PROJECT") != "")
	log.Info("TrustedAssertions utils")

	b64key := os.Getenv("PRV_KEY")
	prvKey := entities.PrivateKeyFromString(b64key)
	err := prvKey.Validate()
	if err != nil {
		log.Errorf("Private key is not valid: %v", err)
		return
	}

	log.Infof("Key: %s", b64key[len(b64key)-8:])

	entity := entities.NewEntity("Mr Tester", *big.NewInt(8376446832743937489))
	entity.MakeCertificate(prvKey)

	u := entity.Uri()
	log.Infof("Certificate URI: %s", u)
	hash := u.Hash()

	dirName := "./testdata"

	err = os.WriteFile(dirName+"/"+hash+".txt", []byte(entity.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}

	statement := statements.NewStatement("The universe exists")
	log.Infof("Statement URI: %s", statement.Uri())
	u = statement.Uri()
	hash = u.Hash()

	err = os.WriteFile(dirName+"/"+hash+".txt", []byte(statement.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}

	assertion := assertions.NewAssertion("IsTrue")
	assertion.Subject = u.String()
	assertion.SetAssertingEntity(entity)
	assertion.MakeJwt(prvKey)
	log.Infof("Assertion URI: %s", assertion.Uri())
	u = assertion.Uri()
	hash = u.Hash()

	err = os.WriteFile(dirName+"/"+hash+".txt", []byte(assertion.Content()), 0777)
	if err != nil {
		log.Errorf("Error writing file %v", err)
	}
}

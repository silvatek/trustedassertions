package auth

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type User struct {
	Id       string `json:"id"`
	PassHash string `json:"passhash"`
}

var DefaultHashCost int = bcrypt.DefaultCost

func (u *User) HashPassword(plaintext string) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), DefaultHashCost)
	if err != nil {
		log.Errorf("Error hashing password: %v", err)
		return
	}
	u.PassHash = base64.StdEncoding.EncodeToString(bytes)
}

func (u *User) CheckHash(plaintext string) bool {
	if u.PassHash == "" {
		return false
	}
	bytes, err := base64.StdEncoding.DecodeString(u.PassHash)
	if err != nil {
		log.Errorf("Error decoding password hash: %v", err)
		return false
	}
	err = bcrypt.CompareHashAndPassword(bytes, []byte(plaintext))
	return err == nil
}

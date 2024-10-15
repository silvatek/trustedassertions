package auth

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
	log "silvatek.uk/trustedassertions/internal/logging"
)

type User struct {
	Id       string `json:"id"`
	PassHash string `json:"passhash"`
	KeyRefs  []KeyRef
}

type KeyRef struct {
	UserId  string `json:"user_id"`
	KeyId   string `json:"key_id"`
	Summary string `json:"summary"`
}

func (u *User) AddKeyRef(keyId string, summary string) {
	if u.KeyRefs == nil {
		u.KeyRefs = make([]KeyRef, 0)
	}
	u.KeyRefs = append(u.KeyRefs, KeyRef{UserId: u.Id, KeyId: keyId, Summary: summary})
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

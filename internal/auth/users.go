package auth

import (
	"bytes"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	log "silvatek.uk/trustedassertions/internal/logging"
	refs "silvatek.uk/trustedassertions/internal/references"
)

const MaxPasskeys = 5

const (
	RoleAuthor        = "Author"
	RoleAdministrator = "Administrator"
)

const (
	UserStatusActive = "Active"
	UserStatusLocked = "Locked"
)

var (
	ErrEmptyPasskeyID  = errors.New("passkey credential id is required")
	ErrTooManyPasskeys = errors.New("user already has the maximum number of passkeys")
	ErrPasskeyExists   = errors.New("passkey already registered")
	ErrPasskeyNotFound = errors.New("passkey not found")
)

type User struct {
	Id         string `json:"id"`
	PassHash   string `json:"passhash"`
	KeyRefs    []KeyRef
	Passkeys   []Passkey   `json:"passkeys"`
	Roles      []string    `json:"roles"`
	Status     string      `json:"status"`
	TrustRoots []TrustRoot `json:"trust_roots,omitempty"`
}

// TrustRoot is an entity the user trusts, with TrustLevel in [0, 1].
type TrustRoot struct {
	EntityUri  string  `json:"entity_uri"` // unadorned hash URI
	TrustLevel float64 `json:"trust_level"`
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

func (u *User) HasKey(keyId string) bool {
	if u.KeyRefs == nil {
		return false
	}

	for _, k := range u.KeyRefs {
		if k.KeyId == keyId {
			return true
		}
	}

	return false
}

func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (u *User) AddRole(role string) {
	if u.HasRole(role) {
		return
	}
	u.Roles = append(u.Roles, role)
}

func (u *User) HasTrustRoot(entity refs.HashUri) bool {
	if entity.IsEmpty() {
		return false
	}
	key := entity.Unadorned()
	for _, r := range u.TrustRoots {
		if refs.UriFromString(r.EntityUri).Unadorned() == key {
			return true
		}
	}
	return false
}

// AddTrustRoot records trust in entity at level if the entity is not already present.
// Empty entities are rejected. Levels are clamped to [0, 1]. Existing entries are not replaced.
func (u *User) AddTrustRoot(entity refs.HashUri, level float64) {
	if entity.IsEmpty() {
		return
	}
	if u.HasTrustRoot(entity) {
		return
	}
	u.TrustRoots = append(u.TrustRoots, TrustRoot{EntityUri: entity.Unadorned(), TrustLevel: clampTrustLevel(level)})
}

func clampTrustLevel(level float64) float64 {
	if level < 0 {
		return 0
	}
	if level > 1 {
		return 1
	}
	return level
}

func (u *User) IsLocked() bool {
	return u.Status == UserStatusLocked
}

func (u *User) SetStatus(status string) {
	u.Status = status
}

func UsersByID(users []User) map[string]User {
	byID := make(map[string]User, len(users))
	for _, user := range users {
		byID[user.Id] = user
	}
	return byID
}

func (u *User) AddPasskey(pk Passkey) error {
	if len(pk.ID) == 0 {
		return ErrEmptyPasskeyID
	}
	if len(u.Passkeys) >= MaxPasskeys {
		return ErrTooManyPasskeys
	}
	if u.passkeyIndex(pk.ID) >= 0 {
		return ErrPasskeyExists
	}
	u.Passkeys = append(u.Passkeys, pk)
	return nil
}

func (u *User) RemovePasskey(credentialID []byte) error {
	if len(credentialID) == 0 {
		return ErrEmptyPasskeyID
	}
	i := u.passkeyIndex(credentialID)
	if i < 0 {
		return ErrPasskeyNotFound
	}
	kept := make([]Passkey, 0, len(u.Passkeys)-1)
	kept = append(kept, u.Passkeys[:i]...)
	kept = append(kept, u.Passkeys[i+1:]...)
	u.Passkeys = kept
	return nil
}

func (u *User) RecordPasskeyUse(update Passkey, usedAt time.Time) error {
	if len(update.ID) == 0 {
		return ErrEmptyPasskeyID
	}
	i := u.passkeyIndex(update.ID)
	if i < 0 {
		return ErrPasskeyNotFound
	}
	u.Passkeys[i].RecordUse(update, usedAt)
	return nil
}

func (u *User) passkeyIndex(credentialID []byte) int {
	for i, pk := range u.Passkeys {
		if bytes.Equal(pk.ID, credentialID) {
			return i
		}
	}
	return -1
}

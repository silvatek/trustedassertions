package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "silvatek.uk/trustedassertions/internal/logging"
)

const ISSUER = "trustedassertions"

const DefaultUserJwtTTL = time.Hour

var ErrUserJwtKeyNotSet = fmt.Errorf("USER_JWT_KEY is not set")

var userJwtKey []byte
var userJwtTTL = DefaultUserJwtTTL

func InitUserJwtFromEnv() error {
	ttl := DefaultUserJwtTTL
	if raw := strings.TrimSpace(os.Getenv("USER_JWT_TTL")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid USER_JWT_TTL %q: %w", raw, err)
		}
		if parsed <= 0 {
			return fmt.Errorf("USER_JWT_TTL must be positive, got %q", raw)
		}
		ttl = parsed
	}

	key := strings.TrimSpace(os.Getenv("USER_JWT_KEY"))
	if key == "" {
		log.Warnf("USER_JWT_KEY is not set; user sessions cannot be created")
		userJwtKey = nil
		userJwtTTL = ttl
		return nil
	}

	return InitUserJwt(key, ttl)
}

func InitUserJwt(key string, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("user JWT key must not be empty")
	}
	if ttl <= 0 {
		ttl = DefaultUserJwtTTL
	}
	userJwtKey = []byte(key)
	userJwtTTL = ttl
	return nil
}

func UserJwtTTL() time.Duration {
	return userJwtTTL
}

func MakeUserJwt(userId string) (string, error) {
	if len(userJwtKey) == 0 {
		return "", ErrUserJwtKeyNotSet
	}
	return makeUserJwt(userId, userJwtKey)
}

func ParseUserJwt(token string) (string, error) {
	return parseUserJwt(token, userJwtKey)
}

func makeUserJwt(userId string, jwtKey []byte) (string, error) {
	ttl := UserJwtTTL()

	claims := jwt.MapClaims{
		"iss": ISSUER,
		"aud": ISSUER,
		"sub": userId,
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().UTC().Add(ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func parseUserJwt(token string, jwtKey []byte) (string, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	}

	userToken, err := jwt.Parse(token, keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(ISSUER),
		jwt.WithAudience(ISSUER),
		jwt.WithIssuedAt(),
	)

	if err != nil {
		return "", err
	}

	return userToken.Claims.GetSubject()
}

package auth

import (
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ISSUER = "trustedassertions"

// Makes a shared key for signing user JWTs for our own consumption
func MakeJwtKey() []byte {
	key := make([]byte, 10)
	rand.Reader.Read(key)
	return key
}

func MakeUserJwt(userId string, jwtKey []byte) (string, error) {
	ttl := 1 * time.Hour

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

func ParseUserJwt(token string, jwtKey []byte) (string, error) {
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

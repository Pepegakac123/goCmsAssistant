package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

func HashPassword(password string) (string, error) {
	hashedPwd, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hashedPwd, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

func MakeJWT(userID int, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "go-cms-assistant",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   fmt.Sprintf("%d", userID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}
func ValidateJWT(tokenString, tokenSecret string) (int, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return 0, err
	}
	userId, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, err
	}
	return userId, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenString := headers.Get("Authorization")
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("token does not exists")
	}
	return tokenString, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	refreshToken := hex.EncodeToString(key)
	return refreshToken, nil
}

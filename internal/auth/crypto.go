package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	issuedAt := jwt.NewNumericDate(now)
	expiresAt := jwt.NewNumericDate(now.Add(expiresIn))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "chirpy", IssuedAt: issuedAt, ExpiresAt: expiresAt, Subject: userID.String()})
	if token == nil {
		return "", fmt.Errorf("error creating a new token")
	}

	signedToken, err := token.SignedString([]byte(tokenSecret))

	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func MakeRefreshToken() (string, error) {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	token := hex.EncodeToString(randBytes)
	return token, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, fmt.Errorf("token is invalid")
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header found")
	}
	splitHeader := strings.Split(authHeader, " ")
	if len(splitHeader) <= 1 || splitHeader[0] != "Bearer" {
		return "", errors.New("invalid authorization header")
	}
	return splitHeader[1], nil
}

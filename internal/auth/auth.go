package auth

import (
	"log"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"time"
	"github.com/google/uuid"
	"fmt"
	"errors"
	"net/http"
	"strings"
	"crypto/rand"
	"encoding/hex"
	"github.com/Tim-Restart/chirpy/internal/database"
	"context"
)

// Function to hash a given password and return the hash
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Print("Error hasing password")
		return "", err
	}

	return string(hashedPassword), nil

}

// Function to check inputed password against users recorded hash

func CheckPasswordHash(hashedPassword, password string) error {
	// Hash the entered password first
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	// Function to make and issue JWT

	claims := jwt.RegisteredClaims{
		// A usual scenario is to set the expiration time relative to the current time
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		Issuer:    "chirpy",
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		log.Print("Error issuing token")
		return "", err
	}
	
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		// Return the key for validation
		return []byte(tokenSecret), nil
	},
)

	if err != nil {
		return uuid.Nil, err
	}

	// Extract claims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}

	// Extract the user ID from the Subject field
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	// Gets the bearer token and does stuff with it
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		log.Print("Error: Authorization header is empty")
		return "", errors.New("authorization header is missing")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Print("Error: Authorization header does not start with 'Bearer '")
		return "", errors.New("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	if strings.TrimSpace(token) == "" {
		log.Print("Error: Token is empty after trimming")
		return "", errors.New("authorization token is empty")
	}

	return token, nil

}

// Makes a refresh token
func MakeRefreshToken() (string, error) {
	key := make([]byte, 32) // Makes an empty byte slice to be filled by the rand.Read
	
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	encodedKey := hex.EncodeToString(key)
	return encodedKey, nil
}

func SaveRefreshToken(token string, userID uuid.UUID, dbQueries database.Queries) error {
	ctx := context.Background()

	params := database.SaveRefTokenParams{
		Token:  token,
		UserID: userID,
	}
	err := dbQueries.SaveRefToken(ctx, params)
	if err != nil {
		return err
	}


	return nil
}
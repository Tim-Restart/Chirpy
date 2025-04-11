package auth

import (
	"testing"
	"time"
	"net/http"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"log"
)

func TestMakeAndValidateJWT(t *testing.T) {
	// Create a test user ID
	userID := uuid.New()
	
	// Define a secret
	tokenSecret := "test-secret"
	
	// Define an expiration time
	expiresIn := time.Hour
	
	// Test creating a JWT
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	// Test validating a valid JWT
	extractedID, err := ValidateJWT(token, tokenSecret)
	assert.NoError(t, err)
	assert.Equal(t, userID, extractedID)
	
	// Test validating JWT with wrong secret
	_, err = ValidateJWT(token, "wrong-secret")
	assert.Error(t, err)
	
	// Test validating an expired JWT
	expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour) // negative duration makes it already expired
	assert.NoError(t, err)
	_, err = ValidateJWT(expiredToken, tokenSecret)
	assert.Error(t, err)
	
	// Test validating an invalid token
	_, err = ValidateJWT("invalid-token", tokenSecret)
	assert.Error(t, err)
}


func TestGetBearerToken(t *testing.T) {

	// Create a test header
	headers := http.Header{}
	headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.Rq8IxqeX7eA6GgYxlcHdPFVRNFFZc5rEI3MQTZZbK3I")

	// Create the token that should be extracted
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.Rq8IxqeX7eA6GgYxlcHdPFVRNFFZc5rEI3MQTZZbK3I"
	// Test that it gets the bearer token from the header
	tokenReturned, err := GetBearerToken(headers)
	assert.NoError(t, err)
	assert.Equal(t, token, tokenReturned)
	log.Printf("Returned Token: %v", tokenReturned)
	log.Printf("Token should be: %v", token)
}
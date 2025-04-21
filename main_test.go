// chirpy_test.go
package main

import (
	"testing"
	"net/http"
	"github.com/stretchr/testify/assert"
	"log"
)

func TestMain(t *testing.T) {
	// This test doesn't do anything yet
	// You would add actual tests here for your main package functionality
}

func TestGetAPIKey(t *testing.T) {

	// Create a test header
	headers := http.Header{}
	headers.Add("Authorization", "ApiKey f271c81ff7084ee5b99a5091b42d486e")
	log.Printf("Test Auth Header: %v", headers)
	key := "f271c81ff7084ee5b99a5091b42d486e"
	keyReturned, err := GetAPIKey(headers) 
	assert.NoError(t, err)
	assert.Equal(t, key, keyReturned)
	log.Printf("Returned Token: %v", keyReturned)
	log.Printf("Token should be: %v", key)

}
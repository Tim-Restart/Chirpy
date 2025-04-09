package auth

import (
	"log"
	"golang.org/x/crypto/bcrypt"
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

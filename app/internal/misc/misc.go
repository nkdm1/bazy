package misc

import (
	"crypto/rand"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword takes a plaintext password and securely hashes it using bcrypt.
// It automatically handles generating a unique cryptographically secure salt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// We log this, but return an empty string so that any login attempt
		// against this broken hash will automatically and safely fail.
		log.Printf("[ERROR] Failed to hash password: %v", err)
		return "", err
	}
	return string(hash), nil
}

// CheckPassword securely compares a bcrypt hash against a plaintext password.
// It mitigates timing attacks and automatically extracts the salt from the hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateSessionID() ([]byte, error) {
	sessionID := make([]byte, 32)
	if _, err := rand.Read(sessionID); err != nil {
		log.Printf("[ERROR] unable to generate session id: %v", err)
		return nil, err
	}
	return sessionID, nil
}

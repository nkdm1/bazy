package misc

import (
	"crypto/rand"
	"log"
	"regexp"

	"github.com/nkdm1/bazy/internal/types"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword takes a plaintext password and securely hashes it using bcrypt.
// It automatically handles generating a unique cryptographically secure salt.
func HashPassword(password string) (string, types.ErrorApi) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Failed to hash password: %v", err)
		return "", types.ErrInternalServer
	}
	return string(hash), nil
}

// CheckPassword securely compares a bcrypt hash against a plaintext password.
// It mitigates timing attacks and automatically extracts the salt from the hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken() returns secure random 32 byte slice of bytes
func GenerateToken() ([]byte, types.ErrorApi) {
	sessionID := make([]byte, 32)
	if _, err := rand.Read(sessionID); err != nil {
		log.Printf("[ERROR] unable to generate session id: %v", err)
		return nil, types.ErrInternalServer
	}
	return sessionID, nil
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail() returns `true` if `email` is valid email regex, otherwise returns false.
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

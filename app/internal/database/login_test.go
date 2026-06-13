package database

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/nkdm1/bazy/internal/types"
)

func TestDeleteAuthToken(t *testing.T) {
	db := testDB(t)

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	// Create a real token the same way login does
	tokenHex := "deadbeefdeadbeefdeadbeefdeadbeef"
	plainTokenBytes, _ := hex.DecodeString(tokenHex)
	tokenHashBytes := sha256.Sum256(plainTokenBytes)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	if err := db.CreateAuthToken(userID, tokenHash); err != nil {
		t.Fatalf("failed to create auth token: %v", err)
	}

	t.Run("successfully deletes an existing token", func(t *testing.T) {
		err := db.DeleteAuthToken(tokenHash)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Verify the token is gone: ValidateSession should now return an error
		_, sessionErr := db.ValidateSession(tokenHash)
		if sessionErr == nil {
			t.Fatal("expected session to be invalid after logout, but ValidateSession returned nil error")
		}
	})

	t.Run("returns ErrNotFound when token does not exist", func(t *testing.T) {
		nonExistentHash := "0000000000000000000000000000000000000000000000000000000000000000"
		err := db.DeleteAuthToken(nonExistentHash)
		if err == nil {
			t.Fatal("expected error for non-existent token, got nil")
		}
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

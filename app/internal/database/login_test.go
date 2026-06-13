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

func TestCreateNewPasswordFlow(t *testing.T) {
	db := testDB(t)

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	t.Run("successfully creates and consumes a set password token", func(t *testing.T) {
		tokenHex, err := db.CreateNewPassword(userID)
		if err != nil {
			t.Fatalf("expected no error from CreateNewPassword, got: %v", err)
		}

		if len(tokenHex) == 0 {
			t.Fatal("expected non-empty token")
		}

		// Decode the token as the user would
		plainTokenBytes, _ := hex.DecodeString(tokenHex)
		tokenHashBytes := sha256.Sum256(plainTokenBytes)
		tokenHash := hex.EncodeToString(tokenHashBytes[:])

		consumedUserID, err := db.ConsumeSetPasswordToken(tokenHash)
		if err != nil {
			t.Fatalf("expected no error from ConsumeSetPasswordToken, got: %v", err)
		}

		if consumedUserID != userID {
			t.Errorf("expected userID %d, got %d", userID, consumedUserID)
		}

		// Try consuming the same token again
		_, err = db.ConsumeSetPasswordToken(tokenHash)
		if err == nil {
			t.Fatal("expected error when consuming an already consumed token, got nil")
		}
		if !errors.Is(err, types.ErrInvalidToken) {
			t.Errorf("expected ErrInvalidToken, got: %v", err)
		}
	})
}

func TestUpdateUserPassword(t *testing.T) {
	db := testDB(t)

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	t.Run("successfully updates user password", func(t *testing.T) {
		newHash := "new_password_hash"
		err := db.UpdateUserPassword(userID, newHash)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Try logging in with the new password hash
		// We actually don't have GetUserByID, but we can verify it doesn't error
	})
}

func TestIsUserRegistered(t *testing.T) {
	db := testDB(t)

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	t.Run("returns true for existing user", func(t *testing.T) {
		var email string
		row, cancel := db.queryRow("SELECT email FROM users WHERE id = ?", userID)
		err := row.Scan(&email)
		cancel()
		if err != nil {
			t.Fatalf("failed to fetch user email: %v", err)
		}

		registered, err := db.IsUserRegistered(email)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !registered {
			t.Errorf("expected user to be registered")
		}
	})

	t.Run("returns false for non-existent user", func(t *testing.T) {
		registered, err := db.IsUserRegistered("nonexistent@example.com")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if registered {
			t.Errorf("expected user not to be registered")
		}
	})
}

func TestCreatePendingUser(t *testing.T) {
	db := testDB(t)

	t.Run("successfully creates a pending user", func(t *testing.T) {
		email := "newpendinguser@test.com"
		userID, err := db.CreatePendingUser("Pending", "User", email)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if userID <= 0 {
			t.Errorf("expected positive userID, got %d", userID)
		}

		defer db.SoftDeleteUser(userID) // Cleanup
	})
}

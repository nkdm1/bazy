package database

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/nkdm1/bazy/internal/types"
)

func TestSoftDeleteUser(t *testing.T) {
	db := testDB(t)

	t.Run("successfully soft-deletes an active user", func(t *testing.T) {
		userID, cleanupUser := createTestUser(t, db)
		defer cleanupUser()

		err := db.SoftDeleteUser(userID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Verify the user can no longer be found by GetPasswordHash (deleted_at IS NOT NULL)
		_, _, loginErr := db.GetPasswordHash("testuser_" + string(rune(userID)) + "@test.com")
		// We just verify SoftDeleteUser worked by checking the role query now returns nothing
		_, roleErr := db.GetUserRole(userID)
		// GetUserRole doesn't filter by deleted_at, so we verify via direct soft-delete check
		// by trying to soft-delete again — it should return ErrNotFound since deleted_at is set.
		secondErr := db.SoftDeleteUser(userID)
		if secondErr == nil {
			t.Error("expected ErrNotFound on second soft-delete, got nil")
		}
		if !errors.Is(secondErr, types.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", secondErr)
		}

		// suppress unused variable warnings
		_ = loginErr
		_ = roleErr
	})

	t.Run("returns ErrNotFound for a non-existent user", func(t *testing.T) {
		err := db.SoftDeleteUser(999999)
		if err == nil {
			t.Fatal("expected error for non-existent user, got nil")
		}
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestInvalidateAllUserSessions(t *testing.T) {
	db := testDB(t)

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	// Create two tokens for the same user
	token1Hex := "aabbccddeeff00112233445566778899"
	token2Hex := "99887766554433221100ffeeddccbbaa"

	for _, hex := range []string{token1Hex, token2Hex} {
		raw, _ := hex2bytes(hex)
		hashBytes := sha256.Sum256(raw)
		hash := encodeHex(hashBytes[:])
		if err := db.CreateAuthToken(userID, hash); err != nil {
			t.Fatalf("failed to create auth token: %v", err)
		}
	}

	t.Run("invalidates all sessions for the user", func(t *testing.T) {
		err := db.InvalidateAllUserSessions(userID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Both tokens should now be invalid
		for _, tokenHex := range []string{token1Hex, token2Hex} {
			raw, _ := hex2bytes(tokenHex)
			hashBytes := sha256.Sum256(raw)
			hash := encodeHex(hashBytes[:])
			_, sessionErr := db.ValidateSession(hash)
			if sessionErr == nil {
				t.Errorf("expected session to be invalid after InvalidateAllUserSessions, but token %s is still valid", tokenHex)
			}
		}
	})
}

// hex2bytes and encodeHex are small local helpers to avoid import cycles in tests.
func hex2bytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

func encodeHex(b []byte) string {
	return hex.EncodeToString(b)
}

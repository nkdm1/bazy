package database

import (
	"testing"
)

func TestRateRefereePerformance(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchCompletedID, _, _, cleanupCompleted := createTestMatch(t, db, "completed", -1)
	defer cleanupCompleted()

	matchScheduledID, _, _, cleanupScheduled := createTestMatch(t, db, "scheduled", 2)
	defer cleanupScheduled()

	userID, cleanupUser := createTestUser(t, db)
	defer cleanupUser()

	t.Run("successfully rate referee performance", func(t *testing.T) {
		apiErr := db.RateRefereePerformance(refereeID, matchCompletedID, 5, userID)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
	})

	t.Run("fail to rate if match not completed", func(t *testing.T) {
		apiErr := db.RateRefereePerformance(refereeID, matchScheduledID, 4, userID)
		if apiErr == nil {
			t.Fatal("expected error rating scheduled match, got nil")
		}
	})
}

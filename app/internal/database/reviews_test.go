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

func TestGetRefereeReviews(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID1, _, _, cleanupMatch1 := createTestMatch(t, db, "completed", -2)
	defer cleanupMatch1()

	matchID2, _, _, cleanupMatch2 := createTestMatch(t, db, "completed", -1)
	defer cleanupMatch2()

	db.RateRefereePerformance(refereeID, matchID1, 4, 1)
	db.RateRefereePerformance(refereeID, matchID2, 5, 1)

	res, err := db.GetRefereeReviews(refereeID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.AverageRating != 4.5 {
		t.Errorf("expected average rating 4.5, got %v", res.AverageRating)
	}
	if len(res.Reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(res.Reviews))
	}
}

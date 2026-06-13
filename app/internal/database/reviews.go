package database

import (
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// RateRefereePerformance adds a review for a referee's performance in a specific match.
// It verifies that the match status is 'completed' before inserting the review.
func (db *Database) RateRefereePerformance(refereeID, matchID, rating, createdBy int) types.ErrorApi {
	row, cancel := db.queryRow(`
		SELECT status
		FROM matches
		WHERE id = ?;
	`, matchID)
	defer cancel()

	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		log.Printf("[ERROR]: Database failure while fetching match %d: %v", matchID, err)
		return types.ErrInternalServer
	}

	if status != "completed" {
		return types.ErrInvalidPayload // Or a more specific error like match not finished
	}

	_, err := db.exec(`
		INSERT INTO reviews (referee_id, match_id, rating, created_by)
		VALUES (?, ?, ?, ?);
	`, refereeID, matchID, rating, createdBy)

	if err != nil {
		log.Printf("[ERROR]: Database failure while inserting review: %v", err)
		return types.ErrInternalServer
	}

	return nil
}

type ReviewInfo struct {
	ID        int    `json:"id"`
	RefereeID int    `json:"referee_id"`
	MatchID   int    `json:"match_id"`
	Rating    int    `json:"rating"`
	CreatedAt string `json:"created_at"`
	CreatedBy int    `json:"created_by"`
}

type RefereeReviews struct {
	AverageRating float64      `json:"average_rating"`
	Reviews       []ReviewInfo `json:"reviews"`
}

func (db *Database) GetRefereeReviews(refereeID int) (*RefereeReviews, types.ErrorApi) {
	row, cancel := db.queryRow(`SELECT IFNULL(AVG(rating), 0) FROM reviews WHERE referee_id = ?`, refereeID)
	var avgRating float64
	err := row.Scan(&avgRating)
	cancel()
	if err != nil {
		log.Printf("[ERROR]: Database failure fetching avg rating for referee %d: %v", refereeID, err)
		return nil, types.ErrInternalServer
	}

	rows, cancelRows, err := db.query(`
		SELECT id, match_id, rating, created_at, created_by 
		FROM reviews 
		WHERE referee_id = ?
		ORDER BY created_at DESC
	`, refereeID)
	defer cancelRows()
	if err != nil {
		log.Printf("[ERROR]: Database failure fetching reviews for referee %d: %v", refereeID, err)
		return nil, types.ErrInternalServer
	}

	list := make([]ReviewInfo, 0)
	for rows.Next() {
		var r ReviewInfo
		r.RefereeID = refereeID
		if err := rows.Scan(&r.ID, &r.MatchID, &r.Rating, &r.CreatedAt, &r.CreatedBy); err != nil {
			log.Printf("[ERROR]: Database failure scanning review: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, r)
	}
	rows.Close()

	return &RefereeReviews{
		AverageRating: avgRating,
		Reviews:       list,
	}, nil
}

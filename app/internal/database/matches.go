package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/nkdm1/bazy/internal/types"
)

type Match struct {
	ID             int
	MatchStart     time.Time
	MatchEnd       time.Time
	LevelOfMatch   int
	VenueID        int
	HomeTeamID     int
	AwayTeamID     int
	Status         string
	HomeTeamPoints sql.NullInt64
	AwayTeamPoints sql.NullInt64
}

// GetMatchesForUpcomingWeek queries the 'matches' table for all matches
// scheduled between now and the next 7 days.
func (db *Database) GetMatchesForUpcomingWeek() ([]Match, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT
			id,
			match_start,
			match_end,
			level_of_match,
			venue_id,
			home_team_id,
			away_team_id,
			status,
			home_team_points,
			away_team_points
		FROM matches
		WHERE match_start BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 7 DAY)
			AND status = 'scheduled';
	`)
	defer cancel()

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching upcoming matches: %v", err)
			return nil, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching upcoming matches: %v", err)
			return nil, types.ErrInternalServer
		}
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		if err := rows.Scan(
			&m.ID,
			&m.MatchStart,
			&m.MatchEnd,
			&m.LevelOfMatch,
			&m.VenueID,
			&m.HomeTeamID,
			&m.AwayTeamID,
			&m.Status,
			&m.HomeTeamPoints,
			&m.AwayTeamPoints,
		); err != nil {
			log.Printf("[ERROR]: Failed to scan match row: %v", err)
			return nil, types.ErrInternalServer
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR]: Row iteration error while fetching upcoming matches: %v", err)
		return nil, types.ErrInternalServer
	}

	return matches, nil
}

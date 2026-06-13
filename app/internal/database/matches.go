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
	LevelOfMatch   string
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

// MarkMatchAsCompleted updates the status of a match to 'completed' by its ID.
func (db *Database) MarkMatchAsCompleted(matchID int) types.ErrorApi {
	result, err := db.exec(`
		UPDATE matches
		SET status = 'completed'
		WHERE id = ?
			AND status != 'cancelled';
	`, matchID)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while completing match %d: %v", matchID, err)
			return types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while completing match %d: %v", matchID, err)
			return types.ErrInternalServer
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: Could not retrieve rows affected for match %d: %v", matchID, err)
		return types.ErrInternalServer
	}
	if rowsAffected == 0 {
		return types.ErrNotFound
	}

	return nil
}

// GetTeamIDByName looks up a team ID by the team's name.
func (db *Database) GetTeamIDByName(name string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id FROM teams WHERE name = ?;
	`, name)
	defer cancel()

	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, types.ErrNotFound
		}
		log.Printf("[ERROR]: Database error fetching team %q: %v", name, err)
		return -1, types.ErrInternalServer
	}
	return id, nil
}

// GetVenueIDByName looks up a venue ID by its gym name.
func (db *Database) GetVenueIDByName(gymName string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id FROM venues WHERE gym_name = ?;
	`, gymName)
	defer cancel()

	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, types.ErrNotFound
		}
		log.Printf("[ERROR]: Database error fetching venue %q: %v", gymName, err)
		return -1, types.ErrInternalServer
	}
	return id, nil
}

// CreateMatch inserts a new scheduled match into the matches table.
func (db *Database) CreateMatch(homeTeamID, awayTeamID, venueID int, levelOfMatch string, start, end time.Time) types.ErrorApi {
	_, err := db.exec(`
		INSERT INTO matches (home_team_id, away_team_id, venue_id, level_of_match, match_start, match_end, status)
		VALUES (?, ?, ?, ?, ?, ?, 'scheduled');
	`, homeTeamID, awayTeamID, venueID, levelOfMatch, start, end)
	if err != nil {
		log.Printf("[ERROR]: Database failure inserting match: %v", err)
		return types.ErrInternalServer
	}
	return nil
}


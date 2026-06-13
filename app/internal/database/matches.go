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

type AssignmentDetail struct {
	RefereeName    string `json:"referee_name"`
	RefereeSurname string `json:"referee_surname"`
	Role           string `json:"role"`
}

type UpcomingMatch struct {
	ID             int                `json:"id"`
	MatchStart     time.Time          `json:"match_start"`
	MatchEnd       time.Time          `json:"match_end"`
	LevelOfMatch   string             `json:"level_of_match"`
	VenueGymName   string             `json:"venue_gym_name"`
	HomeTeamName   string             `json:"home_team_name"`
	AwayTeamName   string             `json:"away_team_name"`
	Assignments    []AssignmentDetail `json:"assignments"`
}

// GetUpcomingMatchesWithDetails queries the 'matches' table for all upcoming matches
// scheduled from now on, joining teams, venues, and assignments.
func (db *Database) GetUpcomingMatchesWithDetails() ([]UpcomingMatch, types.ErrorApi) {
	query := `
		SELECT 
			m.id, 
			m.match_start, 
			m.match_end, 
			m.level_of_match,
			v.gym_name,
			ht.name as home_team,
			at.name as away_team
		FROM matches m
		JOIN venues v ON m.venue_id = v.id
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		WHERE m.match_start >= NOW() AND m.status = 'scheduled'
		ORDER BY m.match_start ASC
	`
	rows, cancel, err := db.query(query)
	defer cancel()
	if err != nil {
		log.Printf("[ERROR]: DB error fetching upcoming matches: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var matches []UpcomingMatch
	for rows.Next() {
		var m UpcomingMatch
		if err := rows.Scan(&m.ID, &m.MatchStart, &m.MatchEnd, &m.LevelOfMatch, &m.VenueGymName, &m.HomeTeamName, &m.AwayTeamName); err != nil {
			log.Printf("[ERROR]: DB error scanning upcoming match: %v", err)
			return nil, types.ErrInternalServer
		}
		m.Assignments = []AssignmentDetail{}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR]: Row iteration error: %v", err)
		return nil, types.ErrInternalServer
	}

	if len(matches) == 0 {
		return []UpcomingMatch{}, nil
	}

	assignQuery := `
		SELECT 
			ma.match_id,
			u.name,
			u.surname,
			r.match_role
		FROM match_assignments ma
		JOIN role_in_match r ON ma.role = r.id
		JOIN referees ref ON ma.referee_id = ref.id
		JOIN users u ON ref.user_id = u.id
		WHERE ma.assignment_status = 'accepted' AND ma.match_id IN (
			SELECT id FROM matches WHERE match_start >= NOW() AND status = 'scheduled'
		)
	`
	aRows, aCancel, aErr := db.query(assignQuery)
	defer aCancel()
	if aErr != nil {
		log.Printf("[ERROR]: DB error fetching assignments: %v", aErr)
		return nil, types.ErrInternalServer
	}
	defer aRows.Close()

	assignmentsMap := make(map[int][]AssignmentDetail)
	for aRows.Next() {
		var matchID int
		var ad AssignmentDetail
		if err := aRows.Scan(&matchID, &ad.RefereeName, &ad.RefereeSurname, &ad.Role); err != nil {
			log.Printf("[ERROR]: DB error scanning assignment: %v", err)
			return nil, types.ErrInternalServer
		}
		assignmentsMap[matchID] = append(assignmentsMap[matchID], ad)
	}

	for i := range matches {
		if arr, ok := assignmentsMap[matches[i].ID]; ok {
			matches[i].Assignments = arr
		}
	}

	return matches, nil
}

type CompletedMatch struct {
	ID             int       `json:"id"`
	MatchStart     time.Time `json:"match_start"`
	MatchEnd       time.Time `json:"match_end"`
	LevelOfMatch   string    `json:"level_of_match"`
	VenueGymName   string    `json:"venue_gym_name"`
	HomeTeamName   string    `json:"home_team_name"`
	AwayTeamName   string    `json:"away_team_name"`
	HomeTeamPoints *int      `json:"home_team_points"`
	AwayTeamPoints *int      `json:"away_team_points"`
}

func (db *Database) GetCompletedMatches() ([]CompletedMatch, types.ErrorApi) {
	query := `
		SELECT 
			m.id, 
			m.match_start, 
			m.match_end, 
			m.level_of_match,
			v.gym_name,
			ht.name as home_team,
			at.name as away_team,
			m.home_team_points,
			m.away_team_points
		FROM matches m
		JOIN venues v ON m.venue_id = v.id
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		WHERE m.status = 'completed'
		ORDER BY m.match_start DESC
	`
	rows, cancel, err := db.query(query)
	defer cancel()
	if err != nil {
		log.Printf("[ERROR]: DB error fetching completed matches: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var matches []CompletedMatch
	for rows.Next() {
		var m CompletedMatch
		var homePts, awayPts sql.NullInt64
		if err := rows.Scan(&m.ID, &m.MatchStart, &m.MatchEnd, &m.LevelOfMatch, &m.VenueGymName, &m.HomeTeamName, &m.AwayTeamName, &homePts, &awayPts); err != nil {
			log.Printf("[ERROR]: DB error scanning completed match: %v", err)
			return nil, types.ErrInternalServer
		}
		if homePts.Valid {
			pts := int(homePts.Int64)
			m.HomeTeamPoints = &pts
		}
		if awayPts.Valid {
			pts := int(awayPts.Int64)
			m.AwayTeamPoints = &pts
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR]: Row iteration error: %v", err)
		return nil, types.ErrInternalServer
	}

	if len(matches) == 0 {
		return []CompletedMatch{}, nil
	}
	return matches, nil
}

type MatchDetails struct {
	ID             int                `json:"id"`
	MatchStart     time.Time          `json:"match_start"`
	MatchEnd       time.Time          `json:"match_end"`
	LevelOfMatch   string             `json:"level_of_match"`
	VenueGymName   string             `json:"venue_gym_name"`
	HomeTeamName   string             `json:"home_team_name"`
	AwayTeamName   string             `json:"away_team_name"`
	Status         string             `json:"status"`
	HomeTeamPoints *int               `json:"home_team_points"`
	AwayTeamPoints *int               `json:"away_team_points"`
	Assignments    []AssignmentDetail `json:"assignments"`
}

func (db *Database) GetMatchDetails(matchID int) (MatchDetails, types.ErrorApi) {
	query := `
		SELECT 
			m.id, 
			m.match_start, 
			m.match_end, 
			m.level_of_match,
			m.status,
			v.gym_name,
			ht.name as home_team,
			at.name as away_team,
			m.home_team_points,
			m.away_team_points
		FROM matches m
		JOIN venues v ON m.venue_id = v.id
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		WHERE m.id = ?
	`
	row, cancel := db.queryRow(query, matchID)
	defer cancel()

	var m MatchDetails
	var homePts, awayPts sql.NullInt64
	if err := row.Scan(&m.ID, &m.MatchStart, &m.MatchEnd, &m.LevelOfMatch, &m.Status, &m.VenueGymName, &m.HomeTeamName, &m.AwayTeamName, &homePts, &awayPts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MatchDetails{}, types.ErrNotFound
		}
		log.Printf("[ERROR]: DB error scanning match details: %v", err)
		return MatchDetails{}, types.ErrInternalServer
	}
	if homePts.Valid {
		pts := int(homePts.Int64)
		m.HomeTeamPoints = &pts
	}
	if awayPts.Valid {
		pts := int(awayPts.Int64)
		m.AwayTeamPoints = &pts
	}

	assignQuery := `
		SELECT 
			u.name,
			u.surname,
			r.match_role
		FROM match_assignments ma
		JOIN role_in_match r ON ma.role = r.id
		JOIN referees ref ON ma.referee_id = ref.id
		JOIN users u ON ref.user_id = u.id
		WHERE ma.match_id = ?
	`
	aRows, aCancel, aErr := db.query(assignQuery, matchID)
	defer aCancel()
	if aErr != nil {
		log.Printf("[ERROR]: DB error fetching assignments: %v", aErr)
		return MatchDetails{}, types.ErrInternalServer
	}
	defer aRows.Close()

	m.Assignments = []AssignmentDetail{}
	for aRows.Next() {
		var ad AssignmentDetail
		if err := aRows.Scan(&ad.RefereeName, &ad.RefereeSurname, &ad.Role); err != nil {
			log.Printf("[ERROR]: DB error scanning assignment: %v", err)
			return MatchDetails{}, types.ErrInternalServer
		}
		m.Assignments = append(m.Assignments, ad)
	}

	return m, nil
}

func (db *Database) CancelMatch(matchID int) types.ErrorApi {
	tx, err := db.instance.Begin()
	if err != nil {
		log.Printf("[ERROR]: Failed to start tx for CancelMatch: %v", err)
		return types.ErrInternalServer
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE matches SET status = 'cancelled' WHERE id = ? AND status != 'cancelled'`, matchID)
	if err != nil {
		log.Printf("[ERROR]: DB error cancelling match: %v", err)
		return types.ErrInternalServer
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: DB error getting rows affected: %v", err)
		return types.ErrInternalServer
	}
	if affected == 0 {
		return types.ErrNotFound
	}

	_, err = tx.Exec(`UPDATE match_assignments SET assignment_status = 'cancelled' WHERE match_id = ? AND assignment_status != 'cancelled'`, matchID)
	if err != nil {
		log.Printf("[ERROR]: DB error cancelling match assignments: %v", err)
		return types.ErrInternalServer
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR]: Failed to commit tx for CancelMatch: %v", err)
		return types.ErrInternalServer
	}

	return nil
}

func (db *Database) RescheduleMatch(matchID int, start, end time.Time) types.ErrorApi {
	res, err := db.exec(`UPDATE matches SET match_start = ?, match_end = ? WHERE id = ? AND status = 'scheduled'`, start, end, matchID)
	if err != nil {
		log.Printf("[ERROR]: DB error rescheduling match: %v", err)
		return types.ErrInternalServer
	}
	affected, err := res.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: DB error getting rows affected: %v", err)
		return types.ErrInternalServer
	}
	if affected == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (db *Database) AssignReferee(matchID, refereeID int, role string) types.ErrorApi {
	var roleID int
	row, cancel := db.queryRow(`SELECT id FROM role_in_match WHERE match_role = ?`, role)
	if err := row.Scan(&roleID); err != nil {
		cancel()
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		log.Printf("[ERROR]: DB error getting role_in_match: %v", err)
		return types.ErrInternalServer
	}
	cancel()

	var existing int
	checkRow, checkCancel := db.queryRow(`SELECT 1 FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	err := checkRow.Scan(&existing)
	checkCancel()
	if err == nil {
		return types.ErrConflict
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ERROR]: DB error checking assignment: %v", err)
		return types.ErrInternalServer
	}

	_, err = db.exec(`INSERT INTO match_assignments (match_id, referee_id, role, assignment_status) VALUES (?, ?, ?, 'pending')`, matchID, refereeID, roleID)
	if err != nil {
		log.Printf("[ERROR]: DB error inserting match_assignment: %v", err)
		return types.ErrInternalServer
	}
	return nil
}

func (db *Database) RevokeAssignment(matchID, refereeID int) types.ErrorApi {
	res, err := db.exec(`UPDATE match_assignments SET assignment_status = 'cancelled' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	if err != nil {
		log.Printf("[ERROR]: DB error revoking assignment: %v", err)
		return types.ErrInternalServer
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (db *Database) RespondToAssignment(matchID, refereeID int, accept bool) types.ErrorApi {
	status := "declined"
	if accept {
		status = "accepted"
	}

	res, err := db.exec(`UPDATE match_assignments SET assignment_status = ? WHERE match_id = ? AND referee_id = ? AND assignment_status = 'pending'`, status, matchID, refereeID)
	if err != nil {
		log.Printf("[ERROR]: DB error responding to assignment: %v", err)
		return types.ErrInternalServer
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.ErrNotFound
	}
	return nil
}

type PendingAssignment struct {
	MatchID    int       `json:"match_id"`
	MatchStart time.Time `json:"match_start"`
	MatchEnd   time.Time `json:"match_end"`
	Level      string    `json:"level_of_match"`
	Role       string    `json:"role"`
}

func (db *Database) GetPendingAssignments(refereeID int) ([]PendingAssignment, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT m.id, m.match_start, m.match_end, m.level_of_match, rm.match_role
		FROM match_assignments ma
		JOIN matches m ON ma.match_id = m.id
		JOIN role_in_match rm ON ma.role = rm.id
		WHERE ma.referee_id = ? AND ma.assignment_status = 'pending'
	`, refereeID)
	defer cancel()

	if err != nil {
		log.Printf("[ERROR]: DB error fetching pending assignments: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var list []PendingAssignment
	for rows.Next() {
		var p PendingAssignment
		if err := rows.Scan(&p.MatchID, &p.MatchStart, &p.MatchEnd, &p.Level, &p.Role); err != nil {
			log.Printf("[ERROR]: DB error scanning pending assignment: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, p)
	}
	if list == nil {
		list = []PendingAssignment{}
	}
	return list, nil
}

func (db *Database) CancelAcceptedAssignment(matchID, refereeID int) types.ErrorApi {
	res, err := db.exec(`UPDATE match_assignments SET assignment_status = 'cancelled' WHERE match_id = ? AND referee_id = ? AND assignment_status = 'accepted'`, matchID, refereeID)
	if err != nil {
		log.Printf("[ERROR]: DB error canceling assignment: %v", err)
		return types.ErrInternalServer
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (db *Database) GetAcceptedAssignments(refereeID int) ([]PendingAssignment, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT m.id, m.match_start, m.match_end, m.level_of_match, rm.match_role
		FROM match_assignments ma
		JOIN matches m ON ma.match_id = m.id
		JOIN role_in_match rm ON ma.role = rm.id
		WHERE ma.referee_id = ? AND ma.assignment_status = 'accepted'
		ORDER BY m.match_start ASC
	`, refereeID)
	defer cancel()

	if err != nil {
		log.Printf("[ERROR]: DB error fetching accepted assignments: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var list []PendingAssignment
	for rows.Next() {
		var p PendingAssignment
		if err := rows.Scan(&p.MatchID, &p.MatchStart, &p.MatchEnd, &p.Level, &p.Role); err != nil {
			log.Printf("[ERROR]: DB error scanning accepted assignment: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, p)
	}
	if list == nil {
		list = []PendingAssignment{}
	}
	return list, nil
}

func (db *Database) MarkNoShow(matchID, refereeID int) types.ErrorApi {
	res, err := db.exec(`UPDATE match_assignments SET assignment_status = 'noshow' WHERE match_id = ? AND referee_id = ? AND assignment_status = 'accepted'`, matchID, refereeID)
	if err != nil {
		log.Printf("[ERROR]: DB error marking noshow: %v", err)
		return types.ErrInternalServer
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.ErrNotFound
	}
	return nil
}

type MatchAssignmentHistory struct {
	RefereeID int    `json:"referee_id"`
	Role      string `json:"role"`
	Status    string `json:"assignment_status"`
}

func (db *Database) GetMatchAssignmentHistory(matchID int) ([]MatchAssignmentHistory, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT ma.referee_id, rm.match_role, ma.assignment_status
		FROM match_assignments ma
		JOIN role_in_match rm ON ma.role = rm.id
		WHERE ma.match_id = ?
	`, matchID)
	defer cancel()

	if err != nil {
		log.Printf("[ERROR]: DB error fetching assignment history: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var list []MatchAssignmentHistory
	for rows.Next() {
		var h MatchAssignmentHistory
		if err := rows.Scan(&h.RefereeID, &h.Role, &h.Status); err != nil {
			log.Printf("[ERROR]: DB error scanning assignment history: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, h)
	}
	if list == nil {
		list = []MatchAssignmentHistory{}
	}
	return list, nil
}

func (db *Database) SubmitMatchScore(matchID int, refereeID int, homePoints int, awayPoints int) types.ErrorApi {
	roleRow, cancelRole := db.queryRow(`SELECT id FROM role_in_match WHERE match_role = 'crew_chief'`)
	var crewChiefRoleID int
	if err := roleRow.Scan(&crewChiefRoleID); err != nil {
		cancelRole()
		log.Printf("[ERROR]: DB error getting crew_chief role id: %v", err)
		return types.ErrInternalServer
	}
	cancelRole()

	checkRow, cancelCheck := db.queryRow(`
		SELECT ma.id 
		FROM match_assignments ma
		JOIN matches m ON ma.match_id = m.id
		WHERE ma.match_id = ? AND ma.referee_id = ? AND ma.role = ? AND ma.assignment_status = 'accepted'
		AND m.match_end <= NOW()
	`, matchID, refereeID, crewChiefRoleID)
	var dummy int
	if err := checkRow.Scan(&dummy); err != nil {
		cancelCheck()
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[DEBUG]: SubmitMatchScore 403! matchID=%d, refID=%d, roleID=%d", matchID, refereeID, crewChiefRoleID)
			return types.ErrForbidden
		}
		log.Printf("[ERROR]: DB error checking crew chief status: %v", err)
		return types.ErrInternalServer
	}
	cancelCheck()

	res, err := db.exec(`
		UPDATE matches
		SET status = 'completed', home_team_points = ?, away_team_points = ?
		WHERE id = ? AND status != 'completed'
	`, homePoints, awayPoints, matchID)
	if err != nil {
		log.Printf("[ERROR]: DB error updating match: %v", err)
		return types.ErrInternalServer
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.ErrNotFound
	}

	rows, cancelRows, err := db.query(`
		SELECT id FROM match_assignments WHERE match_id = ? AND assignment_status = 'accepted'
	`, matchID)
	defer cancelRows()
	if err != nil {
		log.Printf("[ERROR]: DB error querying accepted assignments: %v", err)
		return types.ErrInternalServer
	}

	var assignmentIDs []int
	for rows.Next() {
		var aid int
		if err := rows.Scan(&aid); err != nil {
			log.Printf("[ERROR]: DB error scanning assignment: %v", err)
			return types.ErrInternalServer
		}
		assignmentIDs = append(assignmentIDs, aid)
	}
	rows.Close()

	for _, aid := range assignmentIDs {
		if err := db.CreatePendingPayout(aid); err != nil {
			log.Printf("[ERROR]: DB error creating pending payout for assignment %d: %v", aid, err)
		}
	}

	return nil
}

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/go-sql-driver/mysql"
)

const baseURL = "http://localhost:8080"

var (
	adminClient   *http.Client
	refereeClient *http.Client
	ref2Client *http.Client
	ref3Client *http.Client
	viewerClient  *http.Client

	// Global state for extracting IDs
	matchID   int
	refereeID int
	ref2ID    int
	ref3ID    int
	payoutID  int
	regToken  string
	bankTxID  string
	bankTx2   string
	bankTx3   string
	payout2ID int
	payout3ID int

	// Lipgloss styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(1, 4).
			MarginBottom(1).
			Align(lipgloss.Center)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				Bold(true)

	reqStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	resStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A1FE")).Bold(true)
)

type Step struct {
	Scene   string
	Desc    string
	Caller  string
	Method  string
	Path    string
	Payload func() interface{}
	RawBody func() string
	MultiPath []string
	Do      func() string // Does the request and extracts state if needed
}

type model struct {
	steps       []Step
	stepIdx     int
	executed    bool
	responseStr string

	width     int
	height    int
	focusNext bool // true if Next is focused, false if Exit is focused
}

func initialModel() model {
	return model{
		steps:     getSteps(),
		stepIdx:   0,
		executed:  false,
		focusNext: true,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			m.focusNext = false
		case "right", "l":
			m.focusNext = true
		case "enter", " ":
			if !m.focusNext {
				return m, tea.Quit
			}
			// Focus is on Next
			if m.stepIdx >= len(m.steps) {
				return m, tea.Quit // Demo over
			}

			if !m.executed {
				// Execute the step
				m.responseStr = m.steps[m.stepIdx].Do()
				m.executed = true
			} else {
				// Move to next step
				m.stepIdx++
				m.executed = false
				m.responseStr = ""
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	header := titleStyle.Width(m.width).Render("BAZY API DEMO")

	if m.stepIdx >= len(m.steps) {
		endText := lipgloss.NewStyle().Align(lipgloss.Center).Width(m.width).MarginTop(2).Render("Demo Completed Successfully!\nPress [Enter] or [q] to exit.")
		return lipgloss.JoinVertical(lipgloss.Left, header, endText)
	}

	step := m.steps[m.stepIdx]
	sceneTitle := lipgloss.NewStyle().Align(lipgloss.Center).Width(m.width).Bold(true).Foreground(lipgloss.Color("#E88388")).Render(fmt.Sprintf("--- %s: %s ---", step.Scene, step.Desc))
	stepTitle := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Step %d", m.stepIdx+1))

	// Prepare Left Panel
	var reqBodyBytes []byte
	if step.RawBody != nil {
		reqBodyBytes = []byte(step.RawBody())
	} else if step.Payload != nil {
		reqBodyBytes, _ = json.MarshalIndent(step.Payload(), "", "  ")
	}

	callerText := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true).Render(fmt.Sprintf("Executing as: %s", step.Caller))
	var reqText string
	if len(step.MultiPath) > 0 {
		var reqs []string
		for _, p := range step.MultiPath {
			reqs = append(reqs, fmt.Sprintf("[REQUEST] %s %s", step.Method, baseURL+p))
		}
		reqText = strings.Join(reqs, "\n")
	} else {
		reqText = fmt.Sprintf("[REQUEST] %s %s", step.Method, baseURL+step.Path)
	}
	leftContent := callerText + "\n" + reqStyle.Render(reqText)
	if len(reqBodyBytes) > 0 {
		leftContent += "\n\nBody:\n" + string(reqBodyBytes)
	}

	// Prepare Right Panel
	rightContent := resStyle.Render("[RESPONSE] Waiting for execution...")
	if m.executed {
		rightContent = resStyle.Render("[RESPONSE]\n") + "\n" + m.responseStr
	}

	panelWidth := (m.width / 2) - 4
	if panelWidth < 10 {
		panelWidth = 10
	}

	leftBox := panelStyle.Width(panelWidth).Height(m.height - 10).Render(leftContent)
	rightBox := panelStyle.Width(panelWidth).Height(m.height - 10).Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "  ", rightBox)

	// Buttons
	exitBtn := buttonStyle.Render("[ Exit ]")
	nextBtn := buttonStyle.Render("[ Next ]")

	if m.focusNext {
		nextBtn = activeButtonStyle.Render("[ Next ]")
	} else {
		exitBtn = activeButtonStyle.Render("[ Exit ]")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, exitBtn, "   ", nextBtn)
	buttonsCentered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(buttons)

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		sceneTitle,
		stepTitle,
		panels,
		buttonsCentered,
	)

	return mainView
}

func main() {
	cleanupDB()
	defer cleanupDB()

	// Pre-flight check
	res, err := http.Get(baseURL + "/status")
	if err != nil || res.StatusCode != 200 {
		fmt.Println("[ERROR] Application is not running or /status endpoint failed.")
		fmt.Println("Please start the application on port 8080 and try running the demo again.")
		cleanupDB()
		os.Exit(1)
	}

	adminJar, _ := cookiejar.New(nil)
	adminClient = &http.Client{Jar: adminJar}

	refereeJar, _ := cookiejar.New(nil)
	refereeClient = &http.Client{Jar: refereeJar}
	ref2Jar, _ := cookiejar.New(nil)
	ref2Client = &http.Client{Jar: ref2Jar}
	ref3Jar, _ := cookiejar.New(nil)
	ref3Client = &http.Client{Jar: ref3Jar}

	viewerJar, _ := cookiejar.New(nil)
	viewerClient = &http.Client{Jar: viewerJar}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		cleanupDB()
		os.Exit(1)
	}
}

func cleanupDB() {
	db, err := sql.Open("mysql", "root:root@tcp(ubuntu:3306)/db?parseTime=true")
	if err != nil {
		return
	}
	defer db.Close()

	queries := []string{
		"SET FOREIGN_KEY_CHECKS = 0",
		"DELETE FROM reviews",
		"DELETE FROM payouts",
		"DELETE FROM match_assignments",
		"DELETE FROM matches",
		"DELETE FROM teams",
		"DELETE FROM venues",
		"DELETE FROM availability",
		"DELETE FROM wages",
		"DELETE FROM licenses",
		"DELETE FROM licenses_names",
		"DELETE FROM set_phone",
		"DELETE FROM referees",
		"DELETE FROM set_password",
		"DELETE FROM set_mail",
		"DELETE FROM auth_tokens",
		"DELETE FROM users WHERE email != 'admin@example.com'",
		"INSERT IGNORE INTO role_in_match (match_role) VALUES ('crew_chief'), ('umpire')",
		"INSERT IGNORE INTO licenses_names (license_name) VALUES ('fiba')",
		"SET FOREIGN_KEY_CHECKS = 1",
	}

	for _, q := range queries {
		db.Exec(q)
	}
}

// =========================================================================
// DEMO LOGIC & STEPS
// =========================================================================

func getSteps() []Step {
	return []Step{
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin logs in to get session cookie",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/login",
			Payload: func() interface{} { return map[string]interface{}{"email": "admin@example.com", "password": "admin"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/login", map[string]interface{}{"email": "admin@example.com", "password": "admin"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates a new Venue",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/venues",
			Payload: func() interface{} { return map[string]interface{}{"gym_name": "Demo Arena", "postcode": "00-001", "city": "Demo City", "street": "Demo St", "street_number": "1"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/venues", map[string]interface{}{"gym_name": "Demo Arena", "postcode": "00-001", "city": "Demo City", "street": "Demo St", "street_number": "1"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates Team Alpha",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/teams",
			Payload: func() interface{} { return map[string]interface{}{"name": "Team Alpha", "city": "Alpha City"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/teams", map[string]interface{}{"name": "Team Alpha", "city": "Alpha City"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates Team Beta",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/teams",
			Payload: func() interface{} { return map[string]interface{}{"name": "Team Beta", "city": "Beta City"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/teams", map[string]interface{}{"name": "Team Beta", "city": "Beta City"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin schedules a match with non-existent venue (Error Demo)",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/matches",
			Payload: func() interface{} { return map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Bad Venue", "match_level": "fiba", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/matches", map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Bad Venue", "match_level": "fiba", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin schedules a new match",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/matches",
			Payload: func() interface{} { return map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Demo Arena", "match_level": "fiba", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/matches", map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Demo Arena", "match_level": "fiba", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"})
			},
		},
		{
			Scene:  "SCENE 1: ADMIN DATA SETUP",
			Desc:   "Admin views upcoming matches to extract Match ID",
			Caller:  "Admin",
			Method: "GET",
			Path:   "/matches/upcoming",
			Do: func() string {
				res := doReq(adminClient, "GET", "/matches/upcoming", nil)
				matchID = extractMatchID(res, "Team Alpha", "Team Beta")
				return res
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin sets wages for the FIBA league (Multiple)",
			Caller:  "Admin",
			Method:  "POST",
			MultiPath: []string{"/admin/wages", "/admin/wages", "/admin/wages", "/admin/wages"},
			RawBody: func() string {
				return "body1:\n{\n  \"fee\": 150,\n  \"match_level\": \"fiba\",\n  \"match_role\": \"crew_chief\"\n}\nbody2:\n{\n  \"fee\": 100,\n  \"match_level\": \"fiba\",\n  \"match_role\": \"umpire\"\n}\nbody3:\n{\n  \"fee\": 200,\n  \"match_level\": \"plk\",\n  \"match_role\": \"crew_chief\"\n}\nbody4:\n{\n  \"fee\": 150,\n  \"match_level\": \"plk\",\n  \"match_role\": \"umpire\"\n}"
			},
			Do: func() string {
				r1 := doReq(adminClient, "POST", "/admin/wages", map[string]interface{}{"match_level": "fiba", "match_role": "crew_chief", "fee": 150.0})
				r2 := doReq(adminClient, "POST", "/admin/wages", map[string]interface{}{"match_level": "fiba", "match_role": "umpire", "fee": 100.0})
				r3 := doReq(adminClient, "POST", "/admin/wages", map[string]interface{}{"match_level": "plk", "match_role": "crew_chief", "fee": 200.0})
				r4 := doReq(adminClient, "POST", "/admin/wages", map[string]interface{}{"match_level": "plk", "match_role": "umpire", "fee": 150.0})
				return r1 + "\n" + r2 + "\n" + r3 + "\n" + r4
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "New user registers an account",
			Caller:  "Normal User (john@referee.com)",
			Method:  "POST",
			Path:    "/register/",
			Payload: func() interface{} { return map[string]interface{}{"name": "John", "surname": "Whistle", "email": "john@referee.com"} },
			Do: func() string {
				res := doReq(viewerClient, "POST", "/register/", map[string]interface{}{"name": "John", "surname": "Whistle", "email": "john@referee.com"})
				var rResp struct {
					Data struct {
						Token string `json:"fake_email_message"`
					} `json:"data"`
				}
				json.Unmarshal([]byte(getRawJSON(res)), &rResp)
				regToken = rResp.Data.Token
				return res
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "New user confirms their email",
			Caller:  "Normal User (john@referee.com)",
			Method:  "POST",
			Path:    "/register/confirm",
			Payload: func() interface{} { return map[string]interface{}{"token": regToken, "new_password": "password"} },
			Do: func() string {
				return doReq(viewerClient, "POST", "/register/confirm", map[string]interface{}{"token": regToken, "new_password": "password"})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Users update their profile and apply to become referees (Silent bulk)",
			Caller:  "User (john@referee.com)",
			Method:  "POST",
			MultiPath: []string{"/user/profile", "/user/applyReferee"},
			RawBody: func() string {
				return "body1:\n{\n  \"city\": \"Ref City\",\n  \"flat_number\": \"\",\n  \"phone\": \"123456789\",\n  \"postcode\": \"00-001\",\n  \"street\": \"Ref St\",\n  \"street_number\": \"1\"\n}\nbody2:\n{}"
			},
			Do: func() string {
				// Register others silently
				res2 := doReq(viewerClient, "POST", "/register/", map[string]interface{}{"name": "Jane", "surname": "Smith", "email": "jane@referee.com"})
				var rResp2 struct{ Data struct{ Token string `json:"fake_email_message"` } `json:"data"` }
				json.Unmarshal([]byte(getRawJSON(res2)), &rResp2)
				doReq(viewerClient, "POST", "/register/confirm", map[string]interface{}{"token": rResp2.Data.Token, "new_password": "password"})

				res3 := doReq(viewerClient, "POST", "/register/", map[string]interface{}{"name": "Mark", "surname": "Doe", "email": "mark@referee.com"})
				var rResp3 struct{ Data struct{ Token string `json:"fake_email_message"` } `json:"data"` }
				json.Unmarshal([]byte(getRawJSON(res3)), &rResp3)
				doReq(viewerClient, "POST", "/register/confirm", map[string]interface{}{"token": rResp3.Data.Token, "new_password": "password"})

				// Log them in
				doReq(ref2Client, "POST", "/login", map[string]interface{}{"email": "jane@referee.com", "password": "password"})
				doReq(ref3Client, "POST", "/login", map[string]interface{}{"email": "mark@referee.com", "password": "password"})
				doReq(viewerClient, "POST", "/login", map[string]interface{}{"email": "john@referee.com", "password": "password"})

				// Update profiles and apply
				doReq(ref2Client, "POST", "/user/profile", map[string]interface{}{"phone": "222222222", "postcode": "00-002", "city": "City", "street": "St", "street_number": "2", "flat_number": ""})
				doReq(ref2Client, "POST", "/user/applyReferee", nil)

				doReq(ref3Client, "POST", "/user/profile", map[string]interface{}{"phone": "333333333", "postcode": "00-003", "city": "City", "street": "St", "street_number": "3", "flat_number": ""})
				doReq(ref3Client, "POST", "/user/applyReferee", nil)
				
				r1 := doReq(viewerClient, "POST", "/user/profile", map[string]interface{}{"phone": "123456789", "postcode": "00-001", "city": "Ref City", "street": "Ref St", "street_number": "1", "flat_number": ""})
				r2 := doReq(viewerClient, "POST", "/user/applyReferee", nil)
				return r1 + "\n" + r2
			},
		},
		{
			Scene:  "SCENE 2: REFEREE ONBOARDING",
			Desc:   "Admin fetches Referee Directory to get Referee ID",
			Caller:  "Admin",
			Method: "GET",
			Path:   "/admin/referee/directory",
			Payload: nil,
			Do: func() string {
				res := doReq(adminClient, "GET", "/admin/referee/directory", nil)
				refereeID = extractRefereeID(res, "john@referee.com")
				ref2ID = extractRefereeID(res, "jane@referee.com")
				ref3ID = extractRefereeID(res, "mark@referee.com")
				return res
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Referee logs in",
			Caller:  "Referee (john@referee.com)",
			Method:  "POST",
			Path:    "/login",
			Payload: func() interface{} { return map[string]interface{}{"email": "john@referee.com", "password": "password"} },
			Do: func() string {
				doReq(ref2Client, "POST", "/login", map[string]interface{}{"email": "jane@referee.com", "password": "password"})
				doReq(ref3Client, "POST", "/login", map[string]interface{}{"email": "mark@referee.com", "password": "password"})
				return doReq(refereeClient, "POST", "/login", map[string]interface{}{"email": "john@referee.com", "password": "password"})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Referee submits external license validation request (Others run silently)",
			Caller:  "Referee (john@referee.com)",
			Method:  "POST",
			Path:    "/referee/license",
			Payload: func() interface{} { return map[string]interface{}{"license_name": "fiba", "license_number": "FIBA-JOHN-001", "accept": true} },
			Do: func() string {
				doReq(ref2Client, "POST", "/referee/license", map[string]interface{}{"license_name": "fiba", "license_number": "FIBA-JANE-002", "accept": true})
				doReq(ref3Client, "POST", "/referee/license", map[string]interface{}{"license_name": "fiba", "license_number": "FIBA-MARK-003", "accept": true})
				return doReq(refereeClient, "POST", "/referee/license", map[string]interface{}{"license_name": "fiba", "license_number": "FIBA-JOHN-001", "accept": true})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Referee sets their availability",
			Caller:  "Referee (john@referee.com)",
			Method:  "POST",
			Path:    "/referee/availability",
			Payload: func() interface{} { return map[string]interface{}{"date": "2026-06-15"} },
			Do: func() string {
				return doReq(refereeClient, "POST", "/referee/availability", map[string]interface{}{"date": "2026-06-15"})
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Admin assigns Referees to the Match",
			Caller:  "Admin",
			Method:  "POST",
			MultiPath: []string{"/admin/match/assign", "/admin/match/assign", "/admin/match/assign"},
			RawBody: func() string {
				return fmt.Sprintf("body1:\n{\n  \"match_id\": %d,\n  \"referee_id\": %d,\n  \"role\": \"crew_chief\"\n}\nbody2:\n{\n  \"match_id\": %d,\n  \"referee_id\": %d,\n  \"role\": \"umpire\"\n}\nbody3:\n{\n  \"match_id\": %d,\n  \"referee_id\": %d,\n  \"role\": \"umpire\"\n}", matchID, refereeID, matchID, ref2ID, matchID, ref3ID)
			},
			Do: func() string {
				r1 := doReq(adminClient, "POST", "/admin/match/assign", map[string]interface{}{"match_id": matchID, "referee_id": refereeID, "role": "crew_chief"})
				r2 := doReq(adminClient, "POST", "/admin/match/assign", map[string]interface{}{"match_id": matchID, "referee_id": ref2ID, "role": "umpire"})
				r3 := doReq(adminClient, "POST", "/admin/match/assign", map[string]interface{}{"match_id": matchID, "referee_id": ref3ID, "role": "umpire"})
				return r1 + "\n" + r2 + "\n" + r3
			},
		},
		{
			Scene:  "SCENE 3: MATCH ASSIGNMENT",
			Desc:   "Referee views pending assignments",
			Caller:  "Referee (john@referee.com)",
			Method: "GET",
			Path:   "/referee/assignments/pending",
			Payload: nil,
			Do: func() string {
				res := doReq(refereeClient, "GET", "/referee/assignments/pending", nil)
				return res
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Referee accepts the assignment",
			Caller:  "Referee (john@referee.com)",
			Method:  "POST",
			Path:    "/referee/assignment/respond",
			Payload: func() interface{} { return map[string]interface{}{"match_id": matchID, "accept": true} },
			Do: func() string {
				doReq(ref2Client, "POST", "/referee/assignment/respond", map[string]interface{}{"match_id": matchID, "accept": true})
				doReq(ref3Client, "POST", "/referee/assignment/respond", map[string]interface{}{"match_id": matchID, "accept": true})
				return doReq(refereeClient, "POST", "/referee/assignment/respond", map[string]interface{}{"match_id": matchID, "accept": true})
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Referee submits final score for the match",
			Caller:  "Referee (john@referee.com)",
			Method:  "POST",
			Path:    "/referee/match/score",
			Payload: func() interface{} { return map[string]interface{}{"match_id": matchID, "home_team_points": 2, "away_team_points": 1} },
			Do: func() string {
				// Fast forward match end time to past so submission is valid
				db, _ := sql.Open("mysql", "root:root@tcp(ubuntu:3306)/db?parseTime=true")
				db.Exec("UPDATE matches SET match_end = NOW() - INTERVAL 1 HOUR WHERE id = ?", matchID)
				db.Close()
				return doReq(refereeClient, "POST", "/referee/match/score", map[string]interface{}{"match_id": matchID, "home_team_points": 2, "away_team_points": 1})
			},
		},
		{
			Scene:   "SCENE 4: NORMAL USER ACTIVITY",
			Desc:    "Normal user logs in",
			Caller:  "Normal User (john@referee.com)",
			Method:  "POST",
			Path:    "/login",
			Payload: func() interface{} { return map[string]interface{}{"email": "bob@fan.com", "password": "password"} },
			Do: func() string {
				res := doReq(viewerClient, "POST", "/register/", map[string]interface{}{"name": "Bob", "surname": "Fan", "email": "bob@fan.com"})
				var rResp struct {
					Data struct {
						Token string `json:"fake_email_message"`
					} `json:"data"`
				}
				json.Unmarshal([]byte(getRawJSON(res)), &rResp)
				doReq(viewerClient, "POST", "/register/confirm", map[string]interface{}{"token": rResp.Data.Token, "new_password": "password"})
				return doReq(viewerClient, "POST", "/login", map[string]interface{}{"email": "bob@fan.com", "password": "password"})
			},
		},
		{
			Scene:  "SCENE 4: NORMAL USER ACTIVITY",
			Desc:   "Normal user queries completed matches",
			Caller:  "Normal User (john@referee.com)",
			Method: "GET",
			Path:   "/matches/completed",
			Do: func() string {
				return doReq(viewerClient, "GET", "/matches/completed", nil)
			},
		},
		{
			Scene:   "SCENE 4: NORMAL USER ACTIVITY",
			Desc:    "Normal user submits a 5-star review for the referee",
			Caller:  "Normal User (john@referee.com)",
			Method:  "POST",
			Path:    "/user/rate",
			Payload: func() interface{} { return map[string]interface{}{"referee_id": refereeID, "match_id": matchID, "rating": 5} },
			Do: func() string {
				return doReq(viewerClient, "POST", "/user/rate", map[string]interface{}{"referee_id": refereeID, "match_id": matchID, "rating": 5})
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin fetches pending payouts",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/payouts/pending",
			Payload: func() interface{} { return map[string]interface{}{"all": true} },
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/payouts/pending", map[string]interface{}{"all": true})
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin marks payouts as sent",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/payouts/sent",
			Payload: func() interface{} { return map[string]interface{}{"referee_ids": []int{refereeID, ref2ID, ref3ID}} },
			Do: func() string {
				res := doReq(adminClient, "POST", "/admin/payouts/sent", map[string]interface{}{"referee_ids": []int{refereeID, ref2ID, ref3ID}})
				var parsed map[string]interface{}
				json.Unmarshal([]byte(getRawJSON(res)), &parsed)
				if data, ok := parsed["data"].([]interface{}); ok && len(data) >= 3 {
					if tx, ok := data[0].(map[string]interface{})["bank_transaction_id"].(string); ok { bankTxID = tx }
					if tx, ok := data[1].(map[string]interface{})["bank_transaction_id"].(string); ok { bankTx2 = tx }
					if tx, ok := data[2].(map[string]interface{})["bank_transaction_id"].(string); ok { bankTx3 = tx }
					if pID, ok := data[0].(map[string]interface{})["payout_id"].(float64); ok { payoutID = int(pID) }
					if pID, ok := data[1].(map[string]interface{})["payout_id"].(float64); ok { payout2ID = int(pID) }
					if pID, ok := data[2].(map[string]interface{})["payout_id"].(float64); ok { payout3ID = int(pID) }
				}
				return res
			},
		},
		{
			Scene:  "SCENE 5: PAYOUT PROCESSING",
			Desc:   "Referee views their payout history",
			Caller:  "Referee (john@referee.com)",
			Method: "GET",
			Path:   "/referee/payouts",
			Do: func() string {
				res := doReq(refereeClient, "GET", "/referee/payouts", nil)
				payoutID = extractPayoutID(res)
				return res
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin confirms bank transfers (2 paid, 1 failed)",
			Caller:  "Admin",
			Method:  "POST",
			Path:    "/admin/payouts/confirm",
			Payload: func() interface{} { 
				return map[string]interface{}{"confirmations": []map[string]interface{}{
					{"payout_id": payoutID, "bank_transaction_id": bankTxID, "status": "paid"},
					{"payout_id": payout2ID, "bank_transaction_id": bankTx2, "status": "paid"},
					{"payout_id": payout3ID, "bank_transaction_id": bankTx3, "status": "failed"},
				}} 
			},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/payouts/confirm", map[string]interface{}{
					"confirmations": []map[string]interface{}{
						{"payout_id": payoutID, "bank_transaction_id": bankTxID, "status": "paid"},
						{"payout_id": payout2ID, "bank_transaction_id": bankTx2, "status": "paid"},
						{"payout_id": payout3ID, "bank_transaction_id": bankTx3, "status": "failed"},
					},
				})
			},
		},
		{
			Scene:  "SCENE 5: PAYOUT PROCESSING",
			Desc:   "Admin reviews the monthly payout report",
			Caller:  "Admin",
			Method: "GET",
			Path:   "/admin/payouts/report?year=2026&month=6",
			Do: func() string {
				return doReq(adminClient, "GET", "/admin/payouts/report?year=2026&month=6", nil)
			},
		},
	}
}

func doReq(client *http.Client, method, path string, bodyObj interface{}) string {
	var reqBody io.Reader
	if bodyObj != nil {
		reqBodyBytes, _ := json.Marshal(bodyObj)
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	req, _ := http.NewRequest(method, baseURL+path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	res, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return fmt.Sprintf("[ERROR] Request failed: %v", err)
	}
	defer res.Body.Close()

	resBodyBytes, _ := io.ReadAll(res.Body)

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, resBodyBytes, "", "  "); err == nil {
		resBodyBytes = formatted.Bytes()
	}

	return fmt.Sprintf("Status: %s\nTime: %s\n\n%s", res.Status, duration, string(resBodyBytes))
}

func getRawJSON(s string) string {
	idx := strings.Index(s, "\n\n")
	if idx != -1 {
		return s[idx+2:]
	}
	return s
}

func extractMatchID(resJSON string, home, away string) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(getRawJSON(resJSON)), &parsed)
	if data, ok := parsed["data"].([]interface{}); ok {
		for _, item := range data {
			match := item.(map[string]interface{})
			if match["home_team_name"] == home && match["away_team_name"] == away {
				return int(match["id"].(float64))
			}
		}
	}
	return 0
}

func extractRefereeID(resJSON string, email string) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(getRawJSON(resJSON)), &parsed)
	if data, ok := parsed["data"].([]interface{}); ok {
		for _, item := range data {
			ref := item.(map[string]interface{})
			if ref["email"] == email {
				return int(ref["id"].(float64))
			}
		}
	}
	return 0
}

func extractPayoutID(resJSON string) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(getRawJSON(resJSON)), &parsed)
	if data, ok := parsed["data"].([]interface{}); ok && len(data) > 0 {
		payout := data[0].(map[string]interface{})
		return int(payout["id"].(float64))
	}
	return 0
}

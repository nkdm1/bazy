package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const baseURL = "http://localhost:8080"

var (
	adminClient   *http.Client
	refereeClient *http.Client
	viewerClient  *http.Client

	// Global state for extracting IDs
	matchID      int
	refereeID    int
	assignmentID int
	payoutID     int

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
	Method  string
	Path    string
	Payload interface{}
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
	sceneTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E88388")).Render(fmt.Sprintf("--- %s ---", step.Scene))
	stepTitle := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Step %d: %s", m.stepIdx+1, step.Desc))

	// Prepare Left Panel
	var reqBodyBytes []byte
	if step.Payload != nil {
		reqBodyBytes, _ = json.MarshalIndent(step.Payload, "", "  ")
	}

	leftContent := reqStyle.Render(fmt.Sprintf("[REQUEST] %s %s", step.Method, baseURL+step.Path))
	if step.Payload != nil {
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
	// Pre-flight check
	res, err := http.Get(baseURL + "/status")
	if err != nil || res.StatusCode != 200 {
		fmt.Println("[ERROR] Application is not running or /status endpoint failed.")
		fmt.Println("Please start the application on port 8080 and try running the demo again.")
		os.Exit(1)
	}

	adminJar, _ := cookiejar.New(nil)
	adminClient = &http.Client{Jar: adminJar}

	refereeJar, _ := cookiejar.New(nil)
	refereeClient = &http.Client{Jar: refereeJar}

	viewerJar, _ := cookiejar.New(nil)
	viewerClient = &http.Client{Jar: viewerJar}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
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
			Method:  "POST",
			Path:    "/login",
			Payload: map[string]interface{}{"email": "admin@example.com", "password": "admin"},
			Do: func() string {
				return doReq(adminClient, "POST", "/login", map[string]interface{}{"email": "admin@example.com", "password": "admin"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates a new Venue",
			Method:  "POST",
			Path:    "/admin/venues",
			Payload: map[string]interface{}{"gym_name": "Demo Arena", "postcode": "00-001", "city": "Demo City", "street": "Demo St", "street_number": "1"},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/venues", map[string]interface{}{"gym_name": "Demo Arena", "postcode": "00-001", "city": "Demo City", "street": "Demo St", "street_number": "1"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates Team Alpha",
			Method:  "POST",
			Path:    "/admin/teams",
			Payload: map[string]interface{}{"name": "Team Alpha", "city": "Alpha City"},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/teams", map[string]interface{}{"name": "Team Alpha", "city": "Alpha City"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin creates Team Beta",
			Method:  "POST",
			Path:    "/admin/teams",
			Payload: map[string]interface{}{"name": "Team Beta", "city": "Beta City"},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/teams", map[string]interface{}{"name": "Team Beta", "city": "Beta City"})
			},
		},
		{
			Scene:   "SCENE 1: ADMIN DATA SETUP",
			Desc:    "Admin schedules a new match",
			Method:  "POST",
			Path:    "/admin/matches",
			Payload: map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Demo Arena", "match_level": "Professional", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/matches", map[string]interface{}{"home_team_name": "Team Alpha", "away_team_name": "Team Beta", "venue_name": "Demo Arena", "match_level": "Professional", "match_start": "2026-06-15T12:00:00Z", "match_end": "2026-06-15T14:00:00Z"})
			},
		},
		{
			Scene:  "SCENE 1: ADMIN DATA SETUP",
			Desc:   "Admin views upcoming matches to extract Match ID",
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
			Desc:    "Admin sets wages for the Professional league",
			Method:  "POST",
			Path:    "/admin/wages",
			Payload: map[string]interface{}{"match_level": "Professional", "match_role": "crew_chief", "fee": 150.0},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/wages", map[string]interface{}{"match_level": "Professional", "match_role": "crew_chief", "fee": 150.0})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "New user registers an account",
			Method:  "POST",
			Path:    "/register/",
			Payload: map[string]interface{}{"first_name": "John", "last_name": "Whistle", "email": "john@referee.com", "password": "password", "confirm_password": "password"},
			Do: func() string {
				return doReq(viewerClient, "POST", "/register/", map[string]interface{}{"first_name": "John", "last_name": "Whistle", "email": "john@referee.com", "password": "password", "confirm_password": "password"})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Admin upgrades the user to a Referee",
			Method:  "POST",
			Path:    "/admin/referee",
			Payload: map[string]interface{}{"email": "john@referee.com", "phone": "123456789", "postcode": "00-001", "city": "Ref City", "street": "Ref St", "street_number": "1", "flat_number": ""},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/referee", map[string]interface{}{"email": "john@referee.com", "phone": "123456789", "postcode": "00-001", "city": "Ref City", "street": "Ref St", "street_number": "1", "flat_number": ""})
			},
		},
		{
			Scene:  "SCENE 2: REFEREE ONBOARDING",
			Desc:   "Admin fetches Referee Directory to get Referee ID",
			Method: "GET",
			Path:   "/admin/referee/directory",
			Do: func() string {
				res := doReq(adminClient, "GET", "/admin/referee/directory", nil)
				refereeID = extractRefereeID(res, "john@referee.com")
				return res
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Referee logs in",
			Method:  "POST",
			Path:    "/login",
			Payload: map[string]interface{}{"email": "john@referee.com", "password": "password"},
			Do: func() string {
				return doReq(refereeClient, "POST", "/login", map[string]interface{}{"email": "john@referee.com", "password": "password"})
			},
		},
		{
			Scene:   "SCENE 2: REFEREE ONBOARDING",
			Desc:    "Referee marks their availability",
			Method:  "POST",
			Path:    "/referee/availability",
			Payload: map[string]interface{}{"date": "2026-06-15", "start_time": "08:00:00", "end_time": "20:00:00", "is_available": true},
			Do: func() string {
				return doReq(refereeClient, "POST", "/referee/availability", map[string]interface{}{"date": "2026-06-15", "start_time": "08:00:00", "end_time": "20:00:00", "is_available": true})
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Admin assigns Referee to the Match",
			Method:  "POST",
			Path:    "/admin/match/assign",
			Payload: map[string]interface{}{"role": "crew_chief"}, // MatchID and RefereeID added dynamically in Do
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/match/assign", map[string]interface{}{"match_id": matchID, "referee_id": refereeID, "role": "crew_chief"})
			},
		},
		{
			Scene:  "SCENE 3: MATCH ASSIGNMENT",
			Desc:   "Referee views pending assignments",
			Method: "GET",
			Path:   "/referee/assignments/pending",
			Do: func() string {
				res := doReq(refereeClient, "GET", "/referee/assignments/pending", nil)
				assignmentID = extractAssignmentID(res, matchID)
				return res
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Referee accepts the assignment",
			Method:  "POST",
			Path:    "/referee/assignment/respond",
			Payload: map[string]interface{}{"status": "accepted"}, // assignment_id added dynamically
			Do: func() string {
				return doReq(refereeClient, "POST", "/referee/assignment/respond", map[string]interface{}{"assignment_id": assignmentID, "status": "accepted"})
			},
		},
		{
			Scene:   "SCENE 3: MATCH ASSIGNMENT",
			Desc:    "Referee submits final score for the match",
			Method:  "POST",
			Path:    "/referee/match/score",
			Payload: map[string]interface{}{"home_score": 2, "away_score": 1}, // match_id added dynamically
			Do: func() string {
				return doReq(refereeClient, "POST", "/referee/match/score", map[string]interface{}{"match_id": matchID, "home_score": 2, "away_score": 1})
			},
		},
		{
			Scene:   "SCENE 4: NORMAL USER ACTIVITY",
			Desc:    "Normal user logs in",
			Method:  "POST",
			Path:    "/login",
			Payload: map[string]interface{}{"email": "bob@fan.com", "password": "password"},
			Do: func() string {
				doReq(viewerClient, "POST", "/register/", map[string]interface{}{"first_name": "Bob", "last_name": "Fan", "email": "bob@fan.com", "password": "password", "confirm_password": "password"})
				return doReq(viewerClient, "POST", "/login", map[string]interface{}{"email": "bob@fan.com", "password": "password"})
			},
		},
		{
			Scene:  "SCENE 4: NORMAL USER ACTIVITY",
			Desc:   "Normal user queries completed matches",
			Method: "GET",
			Path:   "/matches/completed",
			Do: func() string {
				return doReq(viewerClient, "GET", "/matches/completed", nil)
			},
		},
		{
			Scene:   "SCENE 4: NORMAL USER ACTIVITY",
			Desc:    "Normal user submits a 5-star review for the referee",
			Method:  "POST",
			Path:    "/user/rate",
			Payload: map[string]interface{}{"rating": 5}, // match_id, referee_id added
			Do: func() string {
				return doReq(viewerClient, "POST", "/user/rate", map[string]interface{}{"referee_id": refereeID, "match_id": matchID, "rating": 5})
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin fetches pending payouts",
			Method:  "POST",
			Path:    "/admin/payouts/pending",
			Payload: map[string]interface{}{"referee_ids": []int{}}, // dynamically updated
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/payouts/pending", map[string]interface{}{"referee_ids": []int{refereeID}})
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin marks payouts as sent",
			Method:  "POST",
			Path:    "/admin/payouts/sent",
			Payload: map[string]interface{}{"referee_ids": []int{}},
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/payouts/sent", map[string]interface{}{"referee_ids": []int{refereeID}})
			},
		},
		{
			Scene:  "SCENE 5: PAYOUT PROCESSING",
			Desc:   "Referee views their payout history",
			Method: "GET",
			Path:   "/referee/payouts",
			Do: func() string {
				res := doReq(refereeClient, "GET", "/referee/payouts", nil)
				payoutID = extractPayoutID(res, assignmentID)
				return res
			},
		},
		{
			Scene:   "SCENE 5: PAYOUT PROCESSING",
			Desc:    "Admin confirms bank transfer payout",
			Method:  "POST",
			Path:    "/admin/payouts/confirm",
			Payload: map[string]interface{}{"confirmations": []map[string]interface{}{{"bank_transaction_id": "TX_999888777"}}}, // payout_id
			Do: func() string {
				return doReq(adminClient, "POST", "/admin/payouts/confirm", map[string]interface{}{
					"confirmations": []map[string]interface{}{{"payout_id": payoutID, "bank_transaction_id": "TX_999888777"}},
				})
			},
		},
		{
			Scene:  "SCENE 5: PAYOUT PROCESSING",
			Desc:   "Admin reviews the monthly payout report",
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

func extractMatchID(resJSON string, home, away string) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(resJSON), &parsed)
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
	json.Unmarshal([]byte(resJSON), &parsed)
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

func extractAssignmentID(resJSON string, matchID int) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(resJSON), &parsed)
	if data, ok := parsed["data"].([]interface{}); ok {
		for _, item := range data {
			assign := item.(map[string]interface{})
			if int(assign["match_id"].(float64)) == matchID {
				return int(assign["id"].(float64))
			}
		}
	}
	return 0
}

func extractPayoutID(resJSON string, assignID int) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(resJSON), &parsed)
	if data, ok := parsed["data"].([]interface{}); ok {
		for _, item := range data {
			payout := item.(map[string]interface{})
			if int(payout["assignment_id"].(float64)) == assignID {
				return int(payout["id"].(float64))
			}
		}
	}
	return 0
}

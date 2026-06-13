package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

const baseURL = "http://localhost:8080"

func main() {
	// Pre-flight check
	res, err := http.Get(baseURL + "/status")
	if err != nil || res.StatusCode != 200 {
		fmt.Println("[ERROR] Application is not running or /status endpoint failed.")
		fmt.Println("Please start the application on port 8080 and try running the demo again.")
		os.Exit(1)
	}
	fmt.Println("[OK] Application is running. Starting demo...\n")

	adminJar, _ := cookiejar.New(nil)
	adminClient := &http.Client{Jar: adminJar}

	refereeJar, _ := cookiejar.New(nil)
	refereeClient := &http.Client{Jar: refereeJar}

	viewerJar, _ := cookiejar.New(nil)
	viewerClient := &http.Client{Jar: viewerJar}

	fmt.Println("=========================================================================================")
	fmt.Println("                                BAZY APPLICATION DEMO                                    ")
	fmt.Println("=========================================================================================")
	fmt.Println("Press ENTER to proceed to the next step at any point.")
	waitEnter()

	// ----------------------------------------------------
	// SCENE 1: ADMIN DATA SETUP
	// ----------------------------------------------------
	
	sendReq(adminClient, 1, "Admin logs in to get session cookie", "POST", "/login", map[string]interface{}{
		"email":    "admin@example.com",
		"password": "admin",
	})

	sendReq(adminClient, 2, "Admin creates a new Venue", "POST", "/admin/venues", map[string]interface{}{
		"gym_name":      "Demo Arena",
		"postcode":      "00-001",
		"city":          "Demo City",
		"street":        "Demo St",
		"street_number": "1",
	})

	sendReq(adminClient, 3, "Admin creates Team Alpha", "POST", "/admin/teams", map[string]interface{}{
		"name": "Team Alpha",
		"city": "Alpha City",
	})

	sendReq(adminClient, 4, "Admin creates Team Beta", "POST", "/admin/teams", map[string]interface{}{
		"name": "Team Beta",
		"city": "Beta City",
	})

	sendReq(adminClient, 5, "Admin schedules a new match", "POST", "/admin/matches", map[string]interface{}{
		"home_team_name": "Team Alpha",
		"away_team_name": "Team Beta",
		"venue_name":     "Demo Arena",
		"match_level":    "Professional",
		"match_start":    "2026-06-15T12:00:00Z",
		"match_end":      "2026-06-15T14:00:00Z",
	})

	upcomingRes := sendReq(adminClient, 6, "Admin views upcoming matches to extract Match ID", "GET", "/matches/upcoming", nil)
	matchID := extractMatchID(upcomingRes, "Team Alpha", "Team Beta")

	sendReq(adminClient, 7, "Admin sets wages for the Professional league", "POST", "/admin/wages", map[string]interface{}{
		"match_level": "Professional",
		"match_role":  "crew_chief",
		"fee":         150.0,
	})

	// ----------------------------------------------------
	// SCENE 2: REFEREE ONBOARDING
	// ----------------------------------------------------
	
	sendReq(viewerClient, 8, "New user registers an account", "POST", "/register/", map[string]interface{}{
		"first_name":       "John",
		"last_name":        "Whistle",
		"email":            "john@referee.com",
		"password":         "password",
		"confirm_password": "password",
	})

	sendReq(adminClient, 9, "Admin upgrades the user to a Referee", "POST", "/admin/referee", map[string]interface{}{
		"email":         "john@referee.com",
		"phone":         "123456789",
		"postcode":      "00-001",
		"city":          "Ref City",
		"street":        "Ref St",
		"street_number": "1",
		"flat_number":   "",
	})

	dirRes := sendReq(adminClient, 10, "Admin fetches Referee Directory to get Referee ID", "GET", "/admin/referee/directory", nil)
	refereeID := extractRefereeID(dirRes, "john@referee.com")

	sendReq(refereeClient, 11, "Referee logs in", "POST", "/login", map[string]interface{}{
		"email":    "john@referee.com",
		"password": "password",
	})

	sendReq(refereeClient, 12, "Referee marks their availability", "POST", "/referee/availability", map[string]interface{}{
		"date":         "2026-06-15",
		"start_time":   "08:00:00",
		"end_time":     "20:00:00",
		"is_available": true,
	})

	// ----------------------------------------------------
	// SCENE 3: MATCH ASSIGNMENT
	// ----------------------------------------------------

	sendReq(adminClient, 13, "Admin assigns Referee to the Match", "POST", "/admin/match/assign", map[string]interface{}{
		"match_id":   matchID,
		"referee_id": refereeID,
		"role":       "crew_chief",
	})

	pendingRes := sendReq(refereeClient, 14, "Referee views pending assignments", "GET", "/referee/assignments/pending", nil)
	assignmentID := extractAssignmentID(pendingRes, matchID)

	sendReq(refereeClient, 15, "Referee accepts the assignment", "POST", "/referee/assignment/respond", map[string]interface{}{
		"assignment_id": assignmentID,
		"status":        "accepted",
	})

	sendReq(refereeClient, 16, "Referee submits final score for the match", "POST", "/referee/match/score", map[string]interface{}{
		"match_id":   matchID,
		"home_score": 2,
		"away_score": 1,
	})

	// ----------------------------------------------------
	// SCENE 4: NORMAL USER ACTIVITY
	// ----------------------------------------------------

	sendReq(viewerClient, 17, "Normal user logs in", "POST", "/login", map[string]interface{}{
		"email":    "bob@fan.com",
		"password": "password",
	})

	sendReq(viewerClient, 18, "Normal user queries completed matches", "GET", "/matches/completed", nil)
	
	sendReq(viewerClient, 19, "Normal user submits a 5-star review for the referee", "POST", "/user/rate", map[string]interface{}{
		"referee_id": refereeID,
		"match_id":   matchID,
		"rating":     5,
	})

	// ----------------------------------------------------
	// SCENE 5: PAYOUT PROCESSING
	// ----------------------------------------------------

	sendReq(adminClient, 20, "Admin fetches pending payouts", "POST", "/admin/payouts/pending", map[string]interface{}{
		"referee_ids": []int{refereeID},
	})

	sendReq(adminClient, 21, "Admin marks payouts as sent", "POST", "/admin/payouts/sent", map[string]interface{}{
		"referee_ids": []int{refereeID},
	})

	payoutsRes := sendReq(refereeClient, 22, "Referee views their payout history", "GET", "/referee/payouts", nil)
	payoutID := extractPayoutID(payoutsRes, assignmentID)

	sendReq(adminClient, 23, "Admin confirms bank transfer payout", "POST", "/admin/payouts/confirm", map[string]interface{}{
		"confirmations": []map[string]interface{}{
			{
				"payout_id":           payoutID,
				"bank_transaction_id": "TX_999888777",
			},
		},
	})

	sendReq(adminClient, 24, "Admin reviews the monthly payout report", "GET", "/admin/payouts/report?year=2026&month=6", nil)

	fmt.Println("\n=========================================================================================")
	fmt.Println("                               DEMO COMPLETED SUCCESSFULLY!                              ")
	fmt.Println("=========================================================================================")
}

func waitEnter() {
	var input []byte = make([]byte, 1)
	os.Stdin.Read(input)
}

func sendReq(client *http.Client, count int, desc, method, path string, bodyObj interface{}) string {
	var reqBody io.Reader
	var reqBodyBytes []byte

	if bodyObj != nil {
		reqBodyBytes, _ = json.MarshalIndent(bodyObj, "", "  ")
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	leftText := fmt.Sprintf("[%d] %s\n\n[REQUEST] %s %s\n", count, desc, method, baseURL+path)
	if bodyObj != nil {
		leftText += "Body:\n" + string(reqBodyBytes) + "\n"
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Println(leftText)
	fmt.Print("(Press ENTER to send request...)")
	waitEnter()

	req, _ := http.NewRequest(method, baseURL+path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	res, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("[ERROR] Request failed: %v\n", err)
		return ""
	}
	defer res.Body.Close()

	resBodyBytes, _ := io.ReadAll(res.Body)
	
	// Pretty format the response if it's JSON
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, resBodyBytes, "", "  "); err == nil {
		resBodyBytes = formatted.Bytes()
	}

	rightText := fmt.Sprintf("[RESPONSE] Status: %s (Time: %s)\n\nBody:\n%s\n", res.Status, duration, string(resBodyBytes))

	fmt.Println("\n" + splitColumns(leftText, rightText, 55))
	return string(resBodyBytes)
}

func splitColumns(left, right string, width int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}
	
	var sb strings.Builder
	for i := 0; i < maxLen; i++ {
		l := ""
		if i < len(leftLines) {
			l = strings.ReplaceAll(leftLines[i], "\r", "")
		}
		r := ""
		if i < len(rightLines) {
			r = strings.ReplaceAll(rightLines[i], "\r", "")
		}
		
		// Prevent super long lines from breaking the split column design
		if len(l) > width {
			l = l[:width-3] + "..."
		}
		
		pad := width - len(l)
		if pad < 2 {
			pad = 2
		}
		
		sb.WriteString(l)
		sb.WriteString(strings.Repeat(" ", pad))
		sb.WriteString("||   ")
		sb.WriteString(r)
		sb.WriteString("\n")
	}
	return sb.String()
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

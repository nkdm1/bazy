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
	adminJar, _ := cookiejar.New(nil)
	adminClient := &http.Client{Jar: adminJar}

	refereeJar, _ := cookiejar.New(nil)
	refereeClient := &http.Client{Jar: refereeJar}

	userJar, _ := cookiejar.New(nil)
	userClient := &http.Client{Jar: userJar}

	fmt.Println("=======================================")
	fmt.Println("       BAZY APPLICATION DEMO")
	fmt.Println("=======================================")
	fmt.Println("Press ENTER to proceed to the next step at any point.")
	waitEnter()

	// ----------------------------------------------------
	// SCENE 1: ADMIN DATA SETUP
	// ----------------------------------------------------
	fmt.Println("\n--- SCENE 1: ADMIN DATA SETUP ---")

	sendReq(adminClient, "POST", "/login", map[string]interface{}{
		"email":    "admin@example.com",
		"password": "admin",
	})

	venueRes := sendReq(adminClient, "POST", "/admin/venues", map[string]interface{}{
		"gym_name":      "Demo Arena",
		"postcode":      "00-001",
		"city":          "Demo City",
		"street":        "Demo St",
		"street_number": "1",
	})
	venueID := extractID(venueRes)

	teamARes := sendReq(adminClient, "POST", "/admin/teams", map[string]interface{}{
		"name": "Team Alpha",
		"city": "Alpha City",
	})
	teamAID := extractID(teamARes)

	teamBRes := sendReq(adminClient, "POST", "/admin/teams", map[string]interface{}{
		"name": "Team Beta",
		"city": "Beta City",
	})
	teamBID := extractID(teamBRes)

	matchRes := sendReq(adminClient, "POST", "/admin/matches", map[string]interface{}{
		"home_team_id": teamAID,
		"away_team_id": teamBID,
		"venue_id":     venueID,
		"date_time":    "2026-06-15T12:00:00Z",
		"league_level": 1,
	})
	matchID := extractID(matchRes)

	// Admin configures wages for league level 1
	sendReq(adminClient, "POST", "/admin/wages", map[string]interface{}{
		"league_level": 1,
		"role":         "crew_chief",
		"amount":       150.0,
	})

	// ----------------------------------------------------
	// SCENE 2: REFEREE ONBOARDING
	// ----------------------------------------------------
	fmt.Println("\n--- SCENE 2: REFEREE ONBOARDING ---")

	// Create user for referee
	sendReq(userClient, "POST", "/register/", map[string]interface{}{
		"first_name":       "John",
		"last_name":        "Whistle",
		"email":            "john@referee.com",
		"password":         "password",
		"confirm_password": "password",
	})

	// Admin upgrades to referee
	sendReq(adminClient, "POST", "/admin/referee", map[string]interface{}{
		"email":         "john@referee.com",
		"phone":         "123456789",
		"postcode":      "00-001",
		"city":          "Ref City",
		"street":        "Ref St",
		"street_number": "1",
		"flat_number":   "",
	})

	dirRes := sendReq(adminClient, "GET", "/admin/referee/directory", nil)
	refereeID := extractRefereeID(dirRes, "john@referee.com")

	// Referee logs in
	sendReq(refereeClient, "POST", "/login", map[string]interface{}{
		"email":    "john@referee.com",
		"password": "password",
	})

	// Referee adds availability
	sendReq(refereeClient, "POST", "/referee/availability", map[string]interface{}{
		"date":         "2026-06-15",
		"start_time":   "08:00:00",
		"end_time":     "20:00:00",
		"is_available": true,
	})

	// ----------------------------------------------------
	// SCENE 3: MATCH ASSIGNMENT
	// ----------------------------------------------------
	fmt.Println("\n--- SCENE 3: MATCH ASSIGNMENT ---")

	// Admin assigns referee
	sendReq(adminClient, "POST", "/admin/match/assign", map[string]interface{}{
		"match_id":   matchID,
		"referee_id": refereeID,
		"role":       "crew_chief",
	})

	// Referee checks pending assignments
	pendingRes := sendReq(refereeClient, "GET", "/referee/assignments/pending", nil)
	assignmentID := extractAssignmentID(pendingRes, matchID)

	// Referee accepts assignment
	sendReq(refereeClient, "POST", "/referee/assignment/respond", map[string]interface{}{
		"assignment_id": assignmentID,
		"status":        "accepted",
	})

	// Referee submits match score
	sendReq(refereeClient, "POST", "/referee/match/score", map[string]interface{}{
		"match_id":   matchID,
		"home_score": 2,
		"away_score": 1,
	})

	// ----------------------------------------------------
	// SCENE 4: NORMAL USER ACTIVITY
	// ----------------------------------------------------
	fmt.Println("\n--- SCENE 4: NORMAL USER ACTIVITY ---")

	viewerJar, _ := cookiejar.New(nil)
	viewerClient := &http.Client{Jar: viewerJar}

	sendReq(viewerClient, "POST", "/register/", map[string]interface{}{
		"first_name":       "Bob",
		"last_name":        "Fan",
		"email":            "bob@fan.com",
		"password":         "password",
		"confirm_password": "password",
	})

	sendReq(viewerClient, "POST", "/login", map[string]interface{}{
		"email":    "bob@fan.com",
		"password": "password",
	})

	sendReq(viewerClient, "GET", "/matches/completed", nil)

	// User reviews referee
	sendReq(viewerClient, "POST", "/user/rate", map[string]interface{}{
		"referee_id": refereeID,
		"match_id":   matchID,
		"rating":     5,
	})

	// ----------------------------------------------------
	// SCENE 5: PAYOUT PROCESSING (ADMIN)
	// ----------------------------------------------------
	fmt.Println("\n--- SCENE 5: PAYOUT PROCESSING ---")

	// Admin views pending payouts
	sendReq(adminClient, "POST", "/admin/payouts/pending", map[string]interface{}{
		"referee_ids": []int{refereeID},
	})

	// Admin marks payout sent
	sendReq(adminClient, "POST", "/admin/payouts/sent", map[string]interface{}{
		"referee_ids": []int{refereeID},
	})

	// Referee checks payout history (to find payout_id)
	payoutsRes := sendReq(refereeClient, "GET", "/referee/payouts", nil)
	payoutID := extractPayoutID(payoutsRes, assignmentID)

	// Admin confirms payment with bank transaction ID
	sendReq(adminClient, "POST", "/admin/payouts/confirm", map[string]interface{}{
		"confirmations": []map[string]interface{}{
			{
				"payout_id":           payoutID,
				"bank_transaction_id": "TX_999888777",
			},
		},
	})

	// Admin checks monthly payout report
	sendReq(adminClient, "GET", "/admin/payouts/report?year=2026&month=6", nil)

	fmt.Println("\n=======================================")
	fmt.Println("       DEMO COMPLETED SUCCESSFULLY!")
	fmt.Println("=======================================")
}

func waitEnter() {
	var input []byte = make([]byte, 1)
	os.Stdin.Read(input)
}

func sendReq(client *http.Client, method, path string, bodyObj interface{}) string {
	var reqBody io.Reader
	var reqBodyBytes []byte

	if bodyObj != nil {
		reqBodyBytes, _ = json.MarshalIndent(bodyObj, "", "  ")
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	fmt.Printf("\n[REQUEST] %s %s\n", method, baseURL+path)
	if bodyObj != nil {
		fmt.Printf("Body:\n%s\n", string(reqBodyBytes))
	}

	fmt.Print("\n(Press ENTER to send request...)")
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

	// Try formatting as JSON
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, resBodyBytes, "", "  "); err == nil {
		resBodyBytes = formatted.Bytes()
	}

	fmt.Printf("[RESPONSE] Status: %s (Time: %s)\n", res.Status, duration)
	fmt.Printf("Body:\n%s\n", string(resBodyBytes))
	fmt.Println(strings.Repeat("-", 50))

	return string(resBodyBytes)
}

func extractID(resJSON string) int {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(resJSON), &parsed)
	if data, ok := parsed["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(float64); ok {
			return int(id)
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

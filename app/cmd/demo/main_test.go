package main

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

func TestDemoSequence(t *testing.T) {
	// Initialize clients
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

	// Ensure DB is clean before test
	cleanupDB()
	defer cleanupDB()

	model := initialModel()
	steps := model.steps

	for i, step := range steps {
		res := step.Do()
		
		// The `doReq` function returns the response body and status
		// We expect no "error" key or "Status: 4xx/5xx" in the output
		if (strings.Contains(res, `"error"`) || strings.Contains(res, "Status: 4") || strings.Contains(res, "Status: 5")) && step.Desc != "Admin schedules a match with non-existent venue" {
			t.Fatalf("Step %d (%s) failed!\nMethod: %s\nPath: %s\nResponse:\n%s", i+1, step.Desc, step.Method, step.Path, res)
		}
	}
}

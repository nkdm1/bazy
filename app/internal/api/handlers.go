package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"maps"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nkdm1/bazy/internal/misc"
	"github.com/nkdm1/bazy/internal/types"
)

// login authenticates user by email and password, then returns session_id via cookie
func (a *Api) login(w http.ResponseWriter, r *http.Request) {
	// if the request expects data, define it as a structure
	// the field types must be pointers to the expected data type
	// 		so in the example below, if i want the email (which is a string)
	// 		i need to specify *string (which is a pointer to a string)
	// use `payload` as the name for these variables
	payload := new(struct {
		Email    *string `json:"email"`
		Password *string `json:"password"`
	})
	// use the `loadPayload()` function to read the request
	//		and store it's data in the `payload` variable
	// remember to check for errors
	if err := loadPayload(payload, r.Body); err != nil {
		// if the err variable is not nil, we have encountered a failure (error)
		// failures should be written to the user using the `fail()` function
		// read more about the `fail()` function at the bottom of the file
		fail(w, err)
		// return after using the `fail()` function
		return
	}
	// extract the values from the payload struct
	email, password := *payload.Email, *payload.Password

	// from this point the 'business'	logic of the function happens

	// call database via a.Database methods
	// create these methods yourself in internal/database/<filename>.go
	userId, hash, dbErr := a.Database.GetPasswordHash(email) // TODO: change so password is taken from auth token and user id is taken from func GetUserByEmail()
	if dbErr != nil {
		fail(w, dbErr)
		return
	}

	// if your logic (like in the example below, checking if password matches database)
	// 		fails, remember to pass correct types.Err* to fail() function
	//		it depends completely on what the logic is so the responsibility is on you
	// misc functions are just generic logic helper functions, you can create them as you need
	if !misc.CheckPassword(hash, password) {
		fail(w, types.ErrInvalidEmailOrPassword)
		return
	}

	sessionId, genErr := misc.GenerateToken()
	if genErr != nil {
		fail(w, genErr)
		return
	}

	// this is also a generic logic but not wrapped as a misc function
	//		because i think it's too short to declare in misc module
	// i would say if a logic takes more than 5 lines of code, it probably should go to misc
	tokenHashBytes := sha256.Sum256(sessionId)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	if dbErr := a.Database.CreateAuthToken(userId, tokenHash); dbErr != nil {
		fail(w, dbErr)
		return
	}

	// this is how creating and attaching cookies looks like
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    hex.EncodeToString(sessionId),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour),
	}
	http.SetCookie(w, cookie)

	// if you need to return any data to the user, create a `response` struct
	// 		similar to the `payload` struct we have defined at the start of the function
	//
	// response := new(struct {
	// 	Score int    `json:"score"`
	// 	Team  string `json:"team"`
	// })
	// response.Score = 2
	// response.Team = "Majowe Borsuki"
	//
	// this time, the field types must NOT be pointers, just use the expected types

	// end the request with `ok()` function
	// attach a positive http.Status<Name>, write a meaningful response message
	// 		and pass the `response` struct if needed, otherwise pass `nil`
	ok(w, http.StatusOK, "login successful", nil)
}

// status returns 'ok'
func (a *Api) status(w http.ResponseWriter, r *http.Request) {
	ok(w, http.StatusOK, "ok", nil)
}

// logout invalidates the current session by deleting the auth token from the database
// and expiring the session_id cookie on the client side.
func (a *Api) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		fail(w, types.ErrUnauthorized)
		return
	}

	plainTokenBytes, err := hex.DecodeString(cookie.Value)
	if err != nil {
		fail(w, types.ErrUnauthorized)
		return
	}

	tokenHashBytes := sha256.Sum256(plainTokenBytes)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	if dbErr := a.Database.DeleteAuthToken(tokenHash); dbErr != nil {
		fail(w, dbErr)
		return
	}

	// expire the cookie on the client side
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	ok(w, http.StatusOK, "logged out", nil)
}

// deleteAccount soft-deletes the authenticated user's account by setting deleted_at
// to the current timestamp, then invalidates all their active sessions.
func (a *Api) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)

	if err := a.Database.SoftDeleteUser(userId); err != nil {
		fail(w, err)
		return
	}

	// Invalidate all sessions so the user is logged out everywhere immediately.
	if err := a.Database.InvalidateAllUserSessions(userId); err != nil {
		fail(w, err)
		return
	}

	// Expire the current session cookie on the client side.
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	ok(w, http.StatusOK, "account deleted", nil)
}

func (a *Api) requestNewPassword(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)

	tokenHex, err := a.Database.CreateNewPassword(userId)
	if err != nil {
		fail(w, err)
		return
	}
	response := new(struct {
		FakeEmailMessage map[string]any `json:"fake_email_message"`
		NextStep         string         `json:"next_step"`
	})
	response.FakeEmailMessage = map[string]any{
		"token": tokenHex,
	}
	response.NextStep = "/user/changePassword/confirm"

	ok(w, 200, "confirm new password", response)
}

func (a *Api) forgotPassword(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Email *string `json:"email"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	userId, err := a.Database.GetUserByEmail(*payload.Email)
	if err != nil {
		if err == types.ErrInvalidEmailOrPassword {
			fail(w, types.ErrNotFound)
		} else {
			fail(w, err)
		}
		return
	}

	tokenHex, err := a.Database.CreateNewPassword(userId)
	if err != nil {
		fail(w, err)
		return
	}
	
	response := new(struct {
		FakeEmailMessage map[string]any `json:"fake_email_message"`
		NextStep         string         `json:"next_step"`
	})
	response.FakeEmailMessage = map[string]any{
		"token": tokenHex,
	}
	response.NextStep = "/forgotPassword/confirm"

	ok(w, 200, "check your email to reset password", response)
}

// wip
func (a *Api) register(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Email   *string `json:"email"`
		Name    *string `json:"name"`
		Surname *string `json:"surname"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}
	name, surname := *payload.Name, *payload.Surname
	email := *payload.Email
	if !misc.IsValidEmail(email) {
		fail(w, types.ErrInvalidEmailFormat)
		return
	}

	registered, dbErr := a.Database.IsUserRegistered(email)
	if dbErr != nil {
		fail(w, dbErr)
		return
	}
	if registered {
		ok(w, 200, "confirm your email", nil)
		return
	}

	userId, err := a.Database.CreatePendingUser(email, name, surname)
	if err != nil {
		fail(w, err)
		return
	}

	tokenHex, err := a.Database.CreateNewPassword(userId)
	if err != nil {
		fail(w, err)
		return
	}

	response := new(struct {
		FakeEmailMessage string `json:"fake_email_message"`
		NextStep         string `json:"next_step"`
	})
	response.FakeEmailMessage = tokenHex
	response.NextStep = "/register/confirm"

	ok(w, 200, "confirm your email", response)
}

func (a *Api) updatePassword(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Token       *string `json:"token"`
		NewPassword *string `json:"new_password"`
	})

	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	tokenHex := *payload.Token
	newPassword := *payload.NewPassword

	plainTokenBytes, err := hex.DecodeString(tokenHex)
	if err != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	tokenHashBytes := sha256.Sum256(plainTokenBytes)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	userId, dbErr := a.Database.ConsumeSetPasswordToken(tokenHash)
	if dbErr != nil {
		fail(w, dbErr)
		return
	}

	hash, bcryptErr := misc.HashPassword(newPassword)
	if bcryptErr != nil {
		fail(w, types.ErrInternalServer)
		return
	}

	if dbErr := a.Database.UpdateUserPassword(userId, hash); dbErr != nil {
		fail(w, dbErr)
		return
	}

	ok(w, 200, "password updated successfully, you can now log in", nil)
}

func (a *Api) addAvailability(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)
	refereeId, err := a.Database.GetRefereeIDByUserID(userId)
	if err != nil {
		fail(w, err)
		return
	}

	payload := new(struct {
		Date *string `json:"date"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	parsedDate, parseErr := time.Parse("2006-01-02", *payload.Date)
	if parseErr != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	if err := a.Database.AddRefereeAvailability(refereeId, parsedDate); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "availability added", nil)
}

func (a *Api) removeAvailability(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)
	refereeId, err := a.Database.GetRefereeIDByUserID(userId)
	if err != nil {
		fail(w, err)
		return
	}

	payload := new(struct {
		Date *string `json:"date"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	parsedDate, parseErr := time.Parse("2006-01-02", *payload.Date)
	if parseErr != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	if err := a.Database.RemoveRefereeAvailability(refereeId, parsedDate); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "availability removed", nil)
}


func (a *Api) rateRefereePerformance(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)

	payload := new(struct {
		RefereeID *int `json:"referee_id"`
		MatchID   *int `json:"match_id"`
		Rating    *int `json:"rating"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.RateRefereePerformance(*payload.RefereeID, *payload.MatchID, *payload.Rating, userId); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "performance rated successfully", nil)
}

// updateWages inserts a new row into the wages table for the given match level
// and role, with valid_from set to today. This preserves historical fee records
// while applying the new fee to future payouts.
func (a *Api) updateWages(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		MatchLevel *string  `json:"match_level"`
		MatchRole  *string  `json:"match_role"`
		Fee        *float64 `json:"fee"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	level := *payload.MatchLevel
	if level != "fiba" && level != "plk" && level != "centralna" && level != "okregowa" && level != "stazysta" {
		fail(w, types.ErrInvalidPayload)
		return
	}

	roleInMatchID, err := a.Database.GetRoleInMatchID(*payload.MatchRole)
	if err != nil {
		fail(w, types.ErrNotFound)
		return
	}

	if err := a.Database.InsertWage(level, roleInMatchID, *payload.Fee); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusCreated, "wages updated", nil)
}

// =========================================================================
// HELPER FUNCTIONS
// =========================================================================

func (a *Api) setRefereeProfile(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Email        *string `json:"email"`
		Phone        *string `json:"phone"`
		Postcode     *string `json:"postcode"`
		City         *string `json:"city"`
		Street       string  `json:"street"`
		StreetNumber *string `json:"street_number"`
		FlatNumber   string  `json:"flat_number"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	postcode := *payload.Postcode
	if len(postcode) != 6 || postcode[2] != '-' {
		fail(w, types.ErrInvalidPayload)
		return
	}

	targetUserID, lookupErr := a.Database.GetUserByEmail(*payload.Email)
	if lookupErr != nil {
		fail(w, types.ErrNotFound)
		return
	}

	if err := a.Database.SetUserAsReferee(targetUserID, *payload.Phone, postcode, *payload.City, payload.Street, *payload.StreetNumber, payload.FlatNumber); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "referee profile created", nil)
}

// createTeam inserts a new team into the teams table.
func (a *Api) createTeam(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Name *string `json:"name"`
		City *string `json:"city"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.CreateTeam(*payload.Name, *payload.City); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusCreated, "team created successfully", nil)
}

// createVenue inserts address details first, then creates a venue tied to it.
func (a *Api) createVenue(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		GymName      *string `json:"gym_name"`
		Postcode     *string `json:"postcode"`
		City         *string `json:"city"`
		Street       string  `json:"street"`
		StreetNumber *string `json:"street_number"`
		FlatNumber   string  `json:"flat_number"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	postcode := *payload.Postcode
	if len(postcode) != 6 || postcode[2] != '-' {
		fail(w, types.ErrInvalidPayload)
		return
	}

	err := a.Database.CreateVenue(*payload.GymName, postcode, *payload.City, payload.Street, *payload.StreetNumber, payload.FlatNumber)
	if err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusCreated, "venue created successfully", nil)
}

// createMatch registers a new match scheduled between two existing teams at a venue.
func (a *Api) createMatch(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		HomeTeamName *string `json:"home_team_name"`
		AwayTeamName *string `json:"away_team_name"`
		VenueName    *string `json:"venue_name"`
		MatchLevel   *string `json:"match_level"`
		MatchStart   *string `json:"match_start"`
		MatchEnd     *string `json:"match_end"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	start, errStart := time.Parse(time.RFC3339, *payload.MatchStart)
	end, errEnd := time.Parse(time.RFC3339, *payload.MatchEnd)
	if errStart != nil || errEnd != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	homeTeamID, err := a.Database.GetTeamIDByName(*payload.HomeTeamName)
	if err != nil {
		fail(w, err)
		return
	}

	awayTeamID, err := a.Database.GetTeamIDByName(*payload.AwayTeamName)
	if err != nil {
		fail(w, err)
		return
	}

	venueID, err := a.Database.GetVenueIDByName(*payload.VenueName)
	if err != nil {
		fail(w, err)
		return
	}

	level := *payload.MatchLevel
	if level != "fiba" && level != "plk" && level != "centralna" && level != "okregowa" && level != "stazysta" {
		fail(w, types.ErrInvalidPayload)
		return
	}

	err = a.Database.CreateMatch(homeTeamID, awayTeamID, venueID, level, start, end)
	if err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusCreated, "match created successfully", nil)
}

// getRefereeDirectory returns a list of all referee profiles with full details.
func (a *Api) getRefereeDirectory(w http.ResponseWriter, r *http.Request) {
	list, err := a.Database.GetRefereeDirectory()
	if err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "referee directory fetched successfully", list)
}

// getRefereeProfile fetches the profile data of the currently authenticated referee.
func (a *Api) getRefereeProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIdKey).(int)

	profile, err := a.Database.GetRefereeProfile(userID)
	if err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "referee profile fetched successfully", profile)
}

// submitLicenseRequest processes a mock license verification and updates database.
func (a *Api) submitLicenseRequest(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		LicenseNumber *string `json:"license_number"`
		LicenseName   *string `json:"license_name"`
		Accept        *bool   `json:"accept"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if !*payload.Accept {
		fail(w, types.ErrInvalidPayload)
		return
	}

	userID := r.Context().Value(UserIdKey).(int)
	refereeID, err := a.Database.GetRefereeIDByUserID(userID)
	if err != nil {
		fail(w, err)
		return
	}

	licenseNameID, err := a.Database.GetLicenseNameID(*payload.LicenseName)
	if err != nil {
		fail(w, err)
		return
	}

	issuedAt := time.Now()
	expireAt := issuedAt.AddDate(1, 0, 0) // Valid for 1 year by default

	err = a.Database.InsertLicense(refereeID, licenseNameID, *payload.LicenseNumber, issuedAt, expireAt)
	if err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusCreated, "license verified and added successfully", nil)
}

// requestNewPhone generates a phone verification token and mock-sends SMS.
func (a *Api) requestNewPhone(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Phone *string `json:"phone"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	userID := r.Context().Value(UserIdKey).(int)
	refereeID, err := a.Database.GetRefereeIDByUserID(userID)
	if err != nil {
		fail(w, err)
		return
	}

	tokenHex, err := a.Database.CreatePhoneChangeToken(refereeID, *payload.Phone)
	if err != nil {
		fail(w, err)
		return
	}

	response := new(struct {
		FakeSMSMessage string `json:"fake_sms_message"`
		NextStep       string `json:"next_step"`
	})
	response.FakeSMSMessage = tokenHex
	response.NextStep = "/referee/setPhone/confirm"

	ok(w, http.StatusOK, "confirm your new phone number", response)
}

// updatePhone consumes the phone verification token and saves new phone.
func (a *Api) updatePhone(w http.ResponseWriter, r *http.Request) {
	payload := new(struct {
		Token *string `json:"token"`
	})
	if err := loadPayload(payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	plainTokenBytes, err := hex.DecodeString(*payload.Token)
	if err != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	tokenHashBytes := sha256.Sum256(plainTokenBytes)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	if err := a.Database.ConsumePhoneChangeToken(tokenHash); err != nil {
		fail(w, err)
		return
	}

	ok(w, http.StatusOK, "phone number updated successfully", nil)
}








// ok writes a successful http response status code `status`
// with `message` attached and, optionally, any `data` provided
//
// `data` has to be in a form acceptable by the json encoder, mostly map[string]any
func ok(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := new(struct {
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	})
	response.Message = message
	if data != nil {
		response.Data = data
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v\n", err)
	}
}

// fail writes an unsuccessful response to user using `err`
//
// fail takes 2 arguments:
//   - w: writes the response to the user
//   - err: sets a informative message on what has happened wrong;
//     you should almost always use `err` for this field;
//     if `err` does not exist, you must declare new basicApiError
//     in types/errors.go and pass it here
func fail(w http.ResponseWriter, err types.ErrorApi) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Code())

	response := map[string]any{
		"error": err.Error(),
	}
	if e, found := errors.AsType[types.ErrorApiWithData](err); found {
		maps.Copy(response, e.ErrorData())
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON error response: %v\n", err)
	}
}

// loadPayload loads the payload from `body` to `dst` structure
func loadPayload(dst any, body io.ReadCloser) types.ErrorApi {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if _, found := errors.AsType[*http.MaxBytesError](err); found {
			return types.ErrPayloadTooLarge
		}
		return types.ErrInvalidJsonBody
	}

	val := reflect.ValueOf(dst)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	var missingFields []string
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		if fieldVal.Kind() == reflect.Pointer {
			if fieldVal.IsNil() ||
				fieldVal.Elem().Kind() == reflect.String && fieldVal.Elem().String() == "" {

				jsonTag := fieldType.Tag.Get("json")
				if jsonTag == "" || jsonTag == "-" {
					jsonTag = fieldType.Name
				} else {
					jsonTag = strings.Split(jsonTag, ",")[0]
				}
				missingFields = append(missingFields, jsonTag)
			}
		}
	}

	if len(missingFields) > 0 {
		return &types.ErrMissingRequiredFields{Fields: missingFields}
	}

	return nil
}

func (a *Api) getUpcomingMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := a.Database.GetUpcomingMatchesWithDetails()
	if err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "upcoming matches", matches)
}

func (a *Api) getCompletedMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := a.Database.GetCompletedMatches()
	if err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "completed matches", matches)
}

func (a *Api) getMatchDetails(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "match_id")
	matchID, err := strconv.Atoi(matchIDStr)
	if err != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	details, apiErr := a.Database.GetMatchDetails(matchID)
	if apiErr != nil {
		fail(w, apiErr)
		return
	}

	ok(w, http.StatusOK, "match details", details)
}

type PayloadCancelMatch struct {
	MatchID *int `json:"match_id"`
}

func (a *Api) cancelMatch(w http.ResponseWriter, r *http.Request) {
	var payload PayloadCancelMatch
	if err := loadPayload(&payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.CancelMatch(*payload.MatchID); err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "match cancelled", nil)
}

type PayloadRescheduleMatch struct {
	MatchID *int       `json:"match_id"`
	Start   *time.Time `json:"match_start"`
	End     *time.Time `json:"match_end"`
}

func (a *Api) rescheduleMatch(w http.ResponseWriter, r *http.Request) {
	var payload PayloadRescheduleMatch
	if err := loadPayload(&payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.RescheduleMatch(*payload.MatchID, *payload.Start, *payload.End); err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "match rescheduled", nil)
}

func (a *Api) searchAvailableReferees(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		fail(w, types.ErrInvalidPayload)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		fail(w, types.ErrInvalidPayload)
		return
	}

	refs, apiErr := a.Database.GetAvailableReferees(date)
	if apiErr != nil {
		fail(w, apiErr)
		return
	}

	ok(w, http.StatusOK, "available referees", refs)
}

type PayloadAssignReferee struct {
	MatchID   *int    `json:"match_id"`
	RefereeID *int    `json:"referee_id"`
	Role      *string `json:"role"`
}

func (a *Api) assignReferee(w http.ResponseWriter, r *http.Request) {
	var payload PayloadAssignReferee
	if err := loadPayload(&payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.AssignReferee(*payload.MatchID, *payload.RefereeID, *payload.Role); err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "referee assigned", nil)
}

type PayloadRevokeAssignment struct {
	MatchID   *int `json:"match_id"`
	RefereeID *int `json:"referee_id"`
}

func (a *Api) revokeAssignment(w http.ResponseWriter, r *http.Request) {
	var payload PayloadRevokeAssignment
	if err := loadPayload(&payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.RevokeAssignment(*payload.MatchID, *payload.RefereeID); err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "assignment revoked", nil)
}

type PayloadRespondToAssignment struct {
	MatchID *int  `json:"match_id"`
	Accept  *bool `json:"accept"`
}

func (a *Api) respondToAssignment(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(UserIdKey).(int)
	refereeID, dbErr := a.Database.GetRefereeIDByUserID(userId)
	if dbErr != nil {
		fail(w, dbErr)
		return
	}

	var payload PayloadRespondToAssignment
	if err := loadPayload(&payload, r.Body); err != nil {
		fail(w, err)
		return
	}

	if err := a.Database.RespondToAssignment(*payload.MatchID, refereeID, *payload.Accept); err != nil {
		fail(w, err)
		return
	}
	ok(w, http.StatusOK, "assignment response recorded", nil)
}

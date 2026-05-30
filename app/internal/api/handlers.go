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
	"strings"
	"time"

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
	userId, hash, dbErr := a.Database.GetPasswordHash(email)
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
	token, err := a.Database.CreateNewPassword(userId)
	if err != nil {
		fail(w, err)
		return
	}
	response := new(struct {
		FakeEmailMessage string `json:"fake_email_message"`
		NextStep         string `json:"next_step"`
	})
	response.FakeEmailMessage = token
	response.NextStep = "/register/confirm"

	ok(w, 200, "confirm your email", response)
}

func (a *Api) registerConfirm(w http.ResponseWriter, r *http.Request) {
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

	userId, dbErr := a.Database.ConsumeRegistrationToken(tokenHash)
	if dbErr != nil {
		fail(w, dbErr)
		return
	}

	hash, bcryptErr := misc.HashPassword(newPassword)
	if bcryptErr != nil {
		fail(w, types.ErrInternalServer)
		return
	}

	if dbErr := a.Database.ActivateUserPassword(userId, hash); dbErr != nil {
		fail(w, dbErr)
		return
	}

	ok(w, 200, "account activated successfully", nil)
}

// =========================================================================
// HELPER FUNCTIONS
// =========================================================================

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

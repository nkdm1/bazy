package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/nkdm1/bazy/internal/misc"
)

// status return 'ok'
func (a *Api) status(w http.ResponseWriter, r *http.Request) {
	ok(w, http.StatusOK, "ok")
}

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
		// failures should be returned to the user using the `fail()` function
		// read more about the `fail()` function at the bottom of the file
		fail(w, http.StatusBadRequest, err.Error())
		// return after using the `fail()` function
		return
	}
	// extract the values from the payload struct
	email, password := *payload.Email, *payload.Password

	// from this point the 'business'	logic of the function happens

	// call database via a.Database methods
	hash, err := a.Database.GetPasswordHash(email)
	if err != nil {
		log.Printf("[ERROR] Database failure during login: %v", err)
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}

	if hash != misc.HashPassword(password) {
		fail(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	sessionID := make([]byte, 32)
	if _, err = rand.Read(sessionID); err != nil {
		log.Printf("[ERROR] Crypto rand read failure: %v", err)
		fail(w, http.StatusInternalServerError,
			fmt.Sprintf("unable to generate the session id: %s", err.Error()))
		return
	}

	// this is how creating and attaching cookies looks like
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    hex.EncodeToString(sessionID),
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
	response := new(struct {
		Score int    `json:"score"`
		Team  string `json:"team"`
	})
	//
	// this time, the field types must NOT be pointers, just use the expected types

	// end the request with `ok()` function
	// attach a positive http.Status<Name>, write a meaningful response message
	// 		and pass the `response` struct if needed
	ok(w, http.StatusOK, "login successful", response)
}

// =========================================================================
// HELPER FUNCTIONS
// =========================================================================

// ok writes a successful http response status code `status`
// with `message` attached and, optionally, any `data` provided
//
// `data` has to be in a form acceptable by the json encoder, for example map[string]string
func ok(w http.ResponseWriter, status int, message string, data ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := new(struct {
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	})
	response.Message = message

	if len(data) > 0 && data[0] != nil {
		response.Data = data[0]
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v\n", err)
	}
}

// fail writes an unsuccessful response with `message` to `w`
// and sets the HTTP response status code to `status`
//
// fail takes 3 arguments:
//   - w (http.ResponseWriter): writes the response to the user
//   - status (int): sets the HTTP Response status codes;
//     use http.Status<Name> for this field, instead of raw numbers;
//     example: use `http.StatusBadRequest` instead of `400`
//   - error (string): sets a informative message on what has happened wrong;
//     you should almost always use err.Error() for this field;
//     if err.Error() does not exist, you must provide a description of the error
func fail(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := new(struct {
		Error string `json:"error"`
	})
	response.Error = err

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode JSON error response: %v\n", err)
	}
}

// loadPayload loads the payload from `r.Body` to `payload` structure
func loadPayload(dst any, body io.ReadCloser) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
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
			if fieldVal.IsNil() {
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
		return fmt.Errorf("missing required fields: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

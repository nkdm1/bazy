package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/nkdm1/bazy/internal/types"
)

type contextKey string

const UserIdKey contextKey = "userId"

func (a *Api) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			fail(w, types.ErrUnauthorized)
			return
		}

		tokenHex := cookie.Value
		plainTokenBytes, err := hex.DecodeString(tokenHex)
		if err != nil {
			fail(w, types.ErrUnauthorized)
			return
		}

		tokenHashBytes := sha256.Sum256(plainTokenBytes)
		tokenHash := hex.EncodeToString(tokenHashBytes[:])

		userId, dbErr := a.Database.ValidateSession(tokenHash)
		if dbErr != nil {
			fail(w, dbErr)
			return
		}

		ctx := context.WithValue(r.Context(), UserIdKey, userId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminOnly is a middleware that must be chained after authorize.
// It reads the userId already stored in the context and verifies the user
// holds the 'admin' role. Returns 403 Forbidden for any other role.
func (a *Api) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Context().Value(UserIdKey).(int)

		role, dbErr := a.Database.GetUserRole(userId)
		if dbErr != nil {
			fail(w, dbErr)
			return
		}
		if role != "admin" {
			fail(w, types.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Api) limitBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

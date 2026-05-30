package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/nkdm1/bazy/internal/types"
)

type contextKey string

const UserIDKey contextKey = "userId"

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

		ctx := context.WithValue(r.Context(), UserIDKey, userId)

		next.ServeHTTP(w, r.WithContext(ctx))
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

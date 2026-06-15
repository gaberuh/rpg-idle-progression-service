package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
)

type contextKey string

const ctxPlayerID contextKey = "player_id"

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeErr(w, apperr.ErrUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, apperr.ErrUnauthorized
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				writeErr(w, apperr.ErrUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeErr(w, apperr.ErrUnauthorized)
				return
			}

			playerIDStr, _ := claims["player_id"].(string)
			playerID, err := uuid.Parse(playerIDStr)
			if err != nil {
				writeErr(w, apperr.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxPlayerID, playerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PlayerIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxPlayerID).(uuid.UUID)
	return id, ok
}

func InternalSecret(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Secret") != secret {
				writeErr(w, apperr.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeErr(w http.ResponseWriter, err *apperr.AppError) {
	http.Error(w, err.Message, err.StatusCode)
}

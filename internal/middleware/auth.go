package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/andr1an/marc-mcp/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type claimsContextKey string

const ClaimsKey claimsContextKey = "jwt_claims"

func AuthDisabled(next http.Handler) http.Handler {
	return next
}

func AuthJWT(validator *auth.JWTValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			claims, err := validator.Validate(parts[1])
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(ctx context.Context) jwt.MapClaims {
	v := ctx.Value(ClaimsKey)
	claims, _ := v.(jwt.MapClaims)
	return claims
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}

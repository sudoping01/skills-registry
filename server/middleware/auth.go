package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/skillhub/skillhub/server/store"
)

// UserTokenAuth validates the Bearer token in the Authorization header against
// the per-user token stored in the filesystem store. The ":user" URL parameter
// must be present in the route (set by chi).
func UserTokenAuth(s *store.FilesystemStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := chi.URLParam(r, "user")

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized: missing Bearer token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			if !s.ValidateUserToken(user, token) {
				http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

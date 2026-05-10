package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/skillhub/skillhub/server/store"
)

var validUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// RegisterHandler handles POST /api/v1/register.
// Body (form or JSON): username → returns {"username":"...","token":"..."} on 201.
func RegisterHandler(s *store.FilesystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		if username == "" || !validUsername.MatchString(username) {
			http.Error(w, "invalid username: must be lowercase letters, digits, hyphens", http.StatusBadRequest)
			return
		}
		token, err := s.CreateUserToken(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"username":"` + username + `","token":"` + token + `"}`))
	}
}

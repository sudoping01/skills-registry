package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/skillhub/skillhub/internal/registry"
	"github.com/skillhub/skillhub/server/store"
)

func SearchHandler(s *store.FilesystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		skills, err := s.Search(query)
		if err != nil {
			http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		result := registry.SearchResult{
			Total:  len(skills),
			Skills: skills,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

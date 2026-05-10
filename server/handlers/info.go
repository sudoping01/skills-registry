package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skillhub/skillhub/server/store"
)

func InfoHandler(s *store.FilesystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		name := chi.URLParam(r, "name")
		version := r.URL.Query().Get("version")

		var (
			meta interface{}
			err  error
		)

		if version != "" {
			meta, err = s.GetMetaVersion(user, name, version)
		} else {
			meta, err = s.GetMeta(user, name)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meta)
	}
}

package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skillhub/skillhub/server/store"
)

func DownloadHandler(s *store.FilesystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		name := chi.URLParam(r, "name")
		version := r.URL.Query().Get("version")

		data, err := s.GetArchive(user, name, version)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		s.IncrementDownloads(user, name)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename="+name+".skill")
		w.Write(data)
	}
}

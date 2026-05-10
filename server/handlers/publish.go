package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skillhub/skillhub/internal/registry"
	"github.com/skillhub/skillhub/server/store"
)

func PublishHandler(s *store.FilesystemStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		name := chi.URLParam(r, "name")

		var req registry.PublishRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		archiveData, err := base64.StdEncoding.DecodeString(req.Archive)
		if err != nil {
			http.Error(w, "invalid archive encoding", http.StatusBadRequest)
			return
		}

		version := req.Version
		if version == "" {
			version = "1.0.0"
		}

		meta := registry.SkillMeta{
			User:          user,
			Name:          name,
			Version:       version,
			Description:   req.Description,
			License:       req.License,
			Compatibility: req.Compatibility,
			Metadata:      req.Metadata,
			Score:         req.Score,
		}

		if err := s.Save(meta, archiveData); err != nil {
			http.Error(w, "failed to save skill: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := registry.PublishResponse{
			Message: "skill published successfully",
			Skill:   meta,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/skillhub/skillhub/internal/registry"
	"github.com/skillhub/skillhub/server/store"
)

// WebHandlers serves the HTML web UI.
type WebHandlers struct {
	store     *store.FilesystemStore
	templates map[string]*template.Template
}

func NewWebHandlers(s *store.FilesystemStore, tmpls map[string]*template.Template) *WebHandlers {
	return &WebHandlers{store: s, templates: tmpls}
}

func (h *WebHandlers) render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Home renders the landing page with stats and recent skills.
func (h *WebHandlers) Home(w http.ResponseWriter, r *http.Request) {
	stats := h.store.Stats()
	skills, _ := h.store.Search("")

	// Show up to 6 most recently published skills (last in slice = latest by dir sort)
	recent := skills
	if len(recent) > 6 {
		recent = recent[len(recent)-6:]
	}
	// Reverse so newest appears first
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	h.render(w, "home", map[string]interface{}{
		"Title":        "",
		"Stats":        stats,
		"RecentSkills": recent,
	})
}

// ExploreSkills renders the paginated skill browse + search page.
func (h *WebHandlers) ExploreSkills(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("q")
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	const pageSize = 20

	all, _ := h.store.Search(keyword)
	total := len(all)
	pages := (total + pageSize - 1) / pageSize

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paged := all[start:end]

	h.render(w, "explore", map[string]interface{}{
		"Title":   "Browse Skills",
		"Skills":  paged,
		"Total":   total,
		"Page":    page,
		"Pages":   pages,
		"Keyword": keyword,
	})
}

// SkillDetail renders the skill detail page.
func (h *WebHandlers) SkillDetail(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	name := chi.URLParam(r, "name")

	meta, err := h.store.GetMeta(user, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Read SKILL.md body from the unpacked archive if available, otherwise empty.
	body := skillBody(h.store, meta)

	h.render(w, "skill", map[string]interface{}{
		"Title": user + "/" + name,
		"Skill": meta,
		"Body":  body,
	})
}

// skillBody attempts to read the SKILL.md body text from the stored archive.
// Returns empty string if unavailable — the detail page handles the absent block.
func skillBody(s *store.FilesystemStore, meta *registry.SkillMeta) string {
	data, err := s.GetSkillMdBody(meta.User, meta.Name, meta.Version)
	if err != nil {
		return ""
	}
	return data
}

// TokenPage renders GET /token/new.
func (h *WebHandlers) TokenPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "token", map[string]interface{}{
		"Title":     "Get API Token",
		"Generated": false,
	})
}

// TokenCreate handles POST /token/new — generates a token and shows it once.
func (h *WebHandlers) TokenCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, "token", map[string]interface{}{
			"Title": "Get API Token",
			"Error": "Invalid form data.",
		})
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))

	token, err := h.store.CreateUserToken(username)
	if err != nil {
		h.render(w, "token", map[string]interface{}{
			"Title":    "Get API Token",
			"Username": username,
			"Error":    err.Error(),
		})
		return
	}

	h.render(w, "token", map[string]interface{}{
		"Title":     "Get API Token",
		"Generated": true,
		"Username":  username,
		"Token":     token,
	})
}

package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/skillhub/skillhub/server/handlers"
	authmiddleware "github.com/skillhub/skillhub/server/middleware"
	"github.com/skillhub/skillhub/server/store"
	skweb "github.com/skillhub/skillhub/server/web"
)

func main() {
	dataDir := os.Getenv("SKILLHUB_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	port := os.Getenv("SKILLHUB_PORT")
	if port == "" {
		port = "8080"
	}

	s, err := store.NewFilesystemStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init store: %v\n", err)
		os.Exit(1)
	}

	tmpls, err := loadTemplates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load templates: %v\n", err)
		os.Exit(1)
	}

	web := handlers.NewWebHandlers(s, tmpls)

	// Sub-FS rooted at "static/" inside the embed for clean URL stripping.
	staticSub, err := fs.Sub(skweb.FS, "static")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open static FS: %v\n", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// ── Static assets ──────────────────────────────────────────────
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// ── Web UI ─────────────────────────────────────────────────────
	r.Get("/", web.Home)
	r.Get("/explore/skills", web.ExploreSkills)
	r.Get("/skills/{user}/{name}", web.SkillDetail)
	r.Get("/token/new", web.TokenPage)
	r.Post("/token/new", web.TokenCreate)

	// ── REST API ───────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/api/v1/search", handlers.SearchHandler(s))
	r.Get("/api/v1/skills/{user}/{name}/info", handlers.InfoHandler(s))
	r.Get("/api/v1/skills/{user}/{name}/download", handlers.DownloadHandler(s))
	r.With(authmiddleware.UserTokenAuth(s)).Post("/api/v1/skills/{user}/{name}", handlers.PublishHandler(s))
	r.Post("/api/v1/register", handlers.RegisterHandler(s))

	fmt.Printf("SkillHub registry running on :%s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

var funcMap = template.FuncMap{
	"scoreTier": func(score int) string {
		switch {
		case score >= 75:
			return "high"
		case score >= 40:
			return "medium"
		default:
			return "low"
		}
	},
	"inc": func(n int) int { return n + 1 },
	"dec": func(n int) int { return n - 1 },
}

func loadTemplates() (map[string]*template.Template, error) {
	pages := []string{"home", "explore", "skill", "token"}
	tmpls := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t, err := template.New("").Funcs(funcMap).ParseFS(skweb.FS,
			"templates/layout.html",
			"templates/"+page+".html",
		)
		if err != nil {
			return nil, fmt.Errorf("parse template %q: %w", page, err)
		}
		tmpls[page] = t
	}
	return tmpls, nil
}

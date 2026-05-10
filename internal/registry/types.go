package registry

// SkillMeta is the registry's stored metadata for a skill version.
type SkillMeta struct {
	User          string            `json:"user"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	License       string            `json:"license"`
	Compatibility string            `json:"compatibility"`
	Metadata      map[string]string `json:"metadata"`
	Dependencies  []string          `json:"dependencies"`
	Score         int               `json:"score"`
	Stars         int               `json:"stars"`
	Downloads     int               `json:"downloads"`
	PublishedAt   string            `json:"published_at"`
	DownloadURL   string            `json:"download_url"`
}

// SearchResult holds a list of matching skills.
type SearchResult struct {
	Total  int         `json:"total"`
	Skills []SkillMeta `json:"skills"`
}

// PublishRequest is the payload sent by the CLI when pushing a skill.
type PublishRequest struct {
	User          string            `json:"user"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	License       string            `json:"license"`
	Compatibility string            `json:"compatibility"`
	Metadata      map[string]string `json:"metadata"`
	Score         int               `json:"score"`
	// Archive is the base64-encoded .skill archive
	Archive string `json:"archive"`
}

// PublishResponse is what the registry returns after a successful publish.
type PublishResponse struct {
	Message string    `json:"message"`
	Skill   SkillMeta `json:"skill"`
}

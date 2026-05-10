package format

import (
	"fmt"
	"strings"
)

// SkillFrontmatter holds the parsed YAML frontmatter from SKILL.md
type SkillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	AllowedTools  string            `yaml:"allowed-tools"`
	Dependencies  []string          `yaml:"dependencies"` // e.g. ["user/html-parser@1.0.0"]
}

// Dependency represents a parsed skill dependency reference.
type Dependency struct {
	User    string
	Name    string
	Version string // empty = latest
}

// ParseDependency parses "user/name" or "user/name@version".
func ParseDependency(dep string) (Dependency, error) {
	atParts := strings.SplitN(dep, "@", 2)
	ref := atParts[0]
	version := ""
	if len(atParts) == 2 {
		version = atParts[1]
	}
	slashParts := strings.SplitN(ref, "/", 2)
	if len(slashParts) != 2 || slashParts[0] == "" || slashParts[1] == "" {
		return Dependency{}, fmt.Errorf("invalid dependency '%s': expected user/name or user/name@version", dep)
	}
	return Dependency{User: slashParts[0], Name: slashParts[1], Version: version}, nil
}

// ParseDependencies parses all dependency strings from a skill's frontmatter.
func ParseDependencies(skill *Skill) ([]Dependency, error) {
	var deps []Dependency
	for _, raw := range skill.Frontmatter.Dependencies {
		dep, err := ParseDependency(raw)
		if err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// Skill represents a fully parsed and validated skill bundle
type Skill struct {
	Frontmatter    SkillFrontmatter
	Body           string // Markdown body after frontmatter
	BodyLines      int    // Number of lines in body
	HasScripts     bool
	HasReferences  bool
	HasAssets      bool
	DirName        string // The directory name the skill was loaded from
	TotalSizeBytes int64
}

// ValidationResult holds all validation errors and warnings
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
	Score    int // 0-100 quality score
}

package format

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	validNameRegex     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
	consecutiveHyphens = regexp.MustCompile(`--`)
)

// Validate checks a parsed Skill against the full specification.
func Validate(skill *Skill) ValidationResult {
	result := ValidationResult{Valid: true}

	// --- Required field: name ---
	if skill.Frontmatter.Name == "" {
		result.addError("name field is required")
	} else {
		if !validNameRegex.MatchString(skill.Frontmatter.Name) {
			result.addError("name must be lowercase letters, digits, and hyphens only")
		}
		if consecutiveHyphens.MatchString(skill.Frontmatter.Name) {
			result.addError("name must not contain consecutive hyphens")
		}
		if len(skill.Frontmatter.Name) > 64 {
			result.addError("name must be 64 characters or fewer")
		}
		if skill.Frontmatter.Name != skill.DirName {
			result.addError(fmt.Sprintf(
				"name field '%s' must match directory name '%s'",
				skill.Frontmatter.Name, skill.DirName,
			))
		}
	}

	// --- Required field: description ---
	if skill.Frontmatter.Description == "" {
		result.addError("description field is required")
	} else {
		if len(skill.Frontmatter.Description) > 1024 {
			result.addError("description must be 1024 characters or fewer")
		}
		if len(skill.Frontmatter.Description) < 20 {
			result.addWarning("description is very short; include what the skill does and when to use it")
		}
	}

	// --- Optional field: compatibility ---
	if len(skill.Frontmatter.Compatibility) > 500 {
		result.addError("compatibility must be 500 characters or fewer")
	}

	// --- Body line count ---
	if skill.BodyLines < 10 {
		result.addWarning("SKILL.md body is very short (< 10 lines); add more instructions")
	}
	if skill.BodyLines > 800 {
		result.addWarning("SKILL.md body exceeds 800 lines; consider moving content to references/")
	}

	// --- Total size ---
	const maxSizeBytes = 2 * 1024 * 1024
	if skill.TotalSizeBytes > maxSizeBytes {
		result.addError(fmt.Sprintf(
			"total skill size %.2f MB exceeds 2 MB limit",
			float64(skill.TotalSizeBytes)/(1024*1024),
		))
	}

	// --- Metadata version ---
	if skill.Frontmatter.Metadata["version"] == "" {
		result.addWarning("metadata.version is not set; consider adding a version (e.g. '1.0.0')")
	}
	if skill.Frontmatter.Metadata["author"] == "" {
		result.addWarning("metadata.author is not set")
	}

	result.Score = computeScore(skill, &result)

	return result
}

func (r *ValidationResult) addError(msg string) {
	r.Errors = append(r.Errors, msg)
	r.Valid = false
}

func (r *ValidationResult) addWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// computeScore produces a 0-100 quality score.
func computeScore(skill *Skill, result *ValidationResult) int {
	score := 100

	score -= len(result.Errors) * 20
	score -= len(result.Warnings) * 5

	if skill.HasScripts {
		score += 5
	}
	if skill.HasReferences {
		score += 5
	}
	if skill.HasAssets {
		score += 3
	}

	if skill.Frontmatter.License != "" {
		score += 3
	}
	if skill.Frontmatter.Compatibility != "" {
		score += 3
	}
	if skill.Frontmatter.Metadata["version"] != "" {
		score += 3
	}

	// Warn if description doesn't start with an uppercase word (heuristic)
	if skill.Frontmatter.Description != "" {
		firstWord := strings.Fields(skill.Frontmatter.Description)[0]
		if firstWord == strings.ToLower(firstWord) {
			result.addWarning("description should start with a verb (e.g. 'Publish', 'Extract', 'Convert')")
			score -= 2
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

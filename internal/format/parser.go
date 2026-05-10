package format

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseSkillDir parses a full skill bundle directory.
func ParseSkillDir(dirPath string) (*Skill, error) {
	dirName := filepath.Base(dirPath)

	skillMdPath := filepath.Join(dirPath, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("SKILL.md not found in %s: %w", dirPath, err)
	}

	frontmatter, body, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	skill := &Skill{
		Frontmatter: frontmatter,
		Body:        body,
		BodyLines:   len(strings.Split(body, "\n")),
		DirName:     dirName,
	}

	skill.HasScripts, _ = dirExists(filepath.Join(dirPath, "scripts"))
	skill.HasReferences, _ = dirExists(filepath.Join(dirPath, "references"))
	skill.HasAssets, _ = dirExists(filepath.Join(dirPath, "assets"))
	skill.TotalSizeBytes, _ = dirSize(dirPath)

	return skill, nil
}

// parseFrontmatter splits SKILL.md content into YAML frontmatter and Markdown body.
func parseFrontmatter(content string) (SkillFrontmatter, string, error) {
	var fm SkillFrontmatter

	if !strings.HasPrefix(content, "---") {
		return fm, content, fmt.Errorf("SKILL.md must start with '---' frontmatter delimiter")
	}

	rest := content[3:] // skip opening ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closingIdx := strings.Index(rest, "\n---")
	if closingIdx == -1 {
		return fm, content, fmt.Errorf("SKILL.md frontmatter closing '---' not found")
	}

	yamlContent := rest[:closingIdx]
	body := rest[closingIdx+4:] // skip \n---
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return fm, body, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	return fm, body, nil
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

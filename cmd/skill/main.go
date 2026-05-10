package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/skillhub/skillhub/internal/format"
	"github.com/skillhub/skillhub/internal/packer"
	"github.com/skillhub/skillhub/internal/registry"
	"github.com/skillhub/skillhub/internal/resolver"
)

const defaultRegistry = "http://localhost:3000"

func getRegistryURL() string {
	if url := os.Getenv("SKILLHUB_REGISTRY"); url != "" {
		return url
	}
	return defaultRegistry
}

func getToken() string {
	if t := os.Getenv("SKILLHUB_TOKEN"); t != "" {
		return t
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".skillhub", "token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// bumpPatch increments the patch segment: "1.2.3" → "1.2.4"
func bumpPatch(version string) (string, error) {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("version '%s' is not semver (expected X.Y.Z)", version)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version '%s'", parts[2])
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, "."), nil
}

func prompt(r *bufio.Reader, msg string) string {
	fmt.Print(msg)
	line, _ := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func promptBool(r *bufio.Reader, msg string) bool {
	answer := strings.ToLower(strings.TrimSpace(prompt(r, msg)))
	return answer == "y" || answer == "yes"
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "skill",
		Short: "SkillHub CLI — manage AI agent skills",
	}

	// ── skill validate <dir> ───────────────────────────────────────────────
	validateCmd := &cobra.Command{
		Use:   "validate <skill-dir>",
		Short: "Validate a skill directory against the specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skill, err := format.ParseSkillDir(args[0])
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}

			result := format.Validate(skill)

			fmt.Printf("Skill: %s\n", skill.Frontmatter.Name)
			fmt.Printf("Score: %d/100\n\n", result.Score)

			if len(result.Errors) > 0 {
				fmt.Println("Errors:")
				for _, e := range result.Errors {
					fmt.Printf("  x %s\n", e)
				}
			}
			if len(result.Warnings) > 0 {
				fmt.Println("Warnings:")
				for _, w := range result.Warnings {
					fmt.Printf("  ! %s\n", w)
				}
			}

			if result.Valid {
				fmt.Println("\n✓ Skill is valid")
			} else {
				fmt.Println("\nSkill validation failed")
				os.Exit(1)
			}
			return nil
		},
	}

	// ── skill push <dir> ───────────────────────────────────────────────────
	var pushUser, pushVersion string
	pushCmd := &cobra.Command{
		Use:   "push <skill-dir>",
		Short: "Publish a skill to the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillDir := args[0]

			skill, err := format.ParseSkillDir(skillDir)
			if err != nil {
				return err
			}
			result := format.Validate(skill)
			if !result.Valid {
				fmt.Println("Skill validation failed. Fix errors before pushing:")
				for _, e := range result.Errors {
					fmt.Printf("  x %s\n", e)
				}
				os.Exit(1)
			}

			user := pushUser
			if user == "" {
				user = os.Getenv("SKILLHUB_USER")
			}
			if user == "" {
				return fmt.Errorf("user required: use --user or set SKILLHUB_USER")
			}

			version := pushVersion
			if version == "" {
				version = skill.Frontmatter.Metadata["version"]
			}
			if version == "" {
				version = "1.0.0"
			}

			archivePath := filepath.Join(os.TempDir(), skill.Frontmatter.Name+".skill")
			if err := packer.Pack(skillDir, archivePath); err != nil {
				return fmt.Errorf("failed to pack skill: %w", err)
			}
			defer os.Remove(archivePath)

			fmt.Printf("Publishing %s/%s@%s...\n", user, skill.Frontmatter.Name, version)

			client := registry.NewClient(getRegistryURL(), getToken())
			resp, err := client.Publish(user, skill.Frontmatter.Name, version, archivePath,
				skill.Frontmatter.Description,
				skill.Frontmatter.License,
				skill.Frontmatter.Compatibility,
				skill.Frontmatter.Metadata,
				result.Score,
			)
			if err != nil {
				return fmt.Errorf("publish failed: %w", err)
			}

			fmt.Printf("✓ Published: %s/%s@%s (score: %d/100)\n",
				resp.Skill.User, resp.Skill.Name, resp.Skill.Version, result.Score)
			return nil
		},
	}
	pushCmd.Flags().StringVar(&pushUser, "user", "", "Registry username")
	pushCmd.Flags().StringVar(&pushVersion, "version", "", "Version to publish (overrides metadata.version)")

	// ── skill install <user/name> ──────────────────────────────────────────
	var installVersion, installDir string
	installCmd := &cobra.Command{
		Use:   "install <user/name>",
		Short: "Install a skill and its dependencies from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid skill reference, expected user/name")
			}
			user, name := parts[0], parts[1]

			version := installVersion
			if strings.Contains(name, "@") {
				nameParts := strings.SplitN(name, "@", 2)
				name = nameParts[0]
				if version == "" {
					version = nameParts[1]
				}
			}

			dest := installDir
			if dest == "" {
				dest = ".skills"
			}

			client := registry.NewClient(getRegistryURL(), getToken())

			fmt.Printf("Resolving dependencies for %s/%s...\n", user, name)
			resolved, err := resolver.Resolve(user, name, version, client)
			if err != nil {
				return fmt.Errorf("dependency resolution failed: %w", err)
			}

			if len(resolved) > 1 {
				fmt.Printf("Installing %d skills (including dependencies):\n", len(resolved))
				for _, r := range resolved {
					fmt.Printf("  → %s/%s@%s\n", r.User, r.Name, r.Version)
				}
			}

			for _, r := range resolved {
				skillDestDir := filepath.Join(dest, r.Name)
				if _, err := os.Stat(skillDestDir); err == nil {
					fmt.Printf("  ✓ %s/%s already installed, skipping\n", r.User, r.Name)
					continue
				}

				archivePath := filepath.Join(os.TempDir(), r.Name+".skill")
				fmt.Printf("  Downloading %s/%s@%s...\n", r.User, r.Name, r.Version)

				if err := client.Download(r.User, r.Name, r.Version, archivePath); err != nil {
					return fmt.Errorf("failed to download %s/%s: %w", r.User, r.Name, err)
				}

				if err := os.MkdirAll(dest, 0o755); err != nil {
					os.Remove(archivePath) //nolint:errcheck
					return err
				}
				if err := packer.Unpack(archivePath, dest); err != nil {
					os.Remove(archivePath) //nolint:errcheck
					return fmt.Errorf("failed to unpack %s/%s: %w", r.User, r.Name, err)
				}
				os.Remove(archivePath) //nolint:errcheck

				fmt.Printf("  ✓ Installed %s/%s@%s\n", r.User, r.Name, r.Version)
			}

			fmt.Printf("\n✓ Done — installed to %s/\n", dest)
			return nil
		},
	}
	installCmd.Flags().StringVar(&installVersion, "version", "", "Version to install")
	installCmd.Flags().StringVar(&installDir, "dir", "", "Install directory (default: .skills/)")

	// ── skill search <query> ───────────────────────────────────────────────
	var searchLicense, searchCompat, searchSort string
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for skills in the registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			url := fmt.Sprintf("%s/api/skillhub/v1/search?q=%s&license=%s&compat=%s&sort=%s",
				getRegistryURL(), query, searchLicense, searchCompat, searchSort)

			resp, err := http.Get(url) //nolint:gosec
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var result registry.SearchResult
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			if result.Total == 0 {
				fmt.Println("No skills found.")
				return nil
			}

			fmt.Printf("Found %d skill(s):\n\n", result.Total)
			for _, s := range result.Skills {
				fmt.Printf("  %s/%s@%s  (score: %d)\n", s.User, s.Name, s.Version, s.Score)
				if s.License != "" {
					fmt.Printf("  License: %s\n", s.License)
				}
				fmt.Printf("  %s\n\n", s.Description)
			}
			return nil
		},
	}
	searchCmd.Flags().StringVar(&searchLicense, "license", "", "Filter by license (e.g. MIT)")
	searchCmd.Flags().StringVar(&searchCompat, "compat", "", "Filter by model compatibility")
	searchCmd.Flags().StringVar(&searchSort, "sort", "", "Sort by: stars, downloads, score, newest")

	// ── skill pull <user/name> ─────────────────────────────────────────────
	var pullOutput string
	pullCmd := &cobra.Command{
		Use:   "pull <user/name>",
		Short: "Fetch the SKILL.md from a published skill",
		Long: `Downloads just the SKILL.md content from the registry.
Prints to stdout by default, or saves to a file with --output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid skill reference, expected user/name")
			}
			user, name := parts[0], parts[1]

			version := ""
			if strings.Contains(name, "@") {
				nameParts := strings.SplitN(name, "@", 2)
				name = nameParts[0]
				version = nameParts[1]
			}

			url := fmt.Sprintf("%s/api/skillhub/v1/skills/%s/%s/readme",
				getRegistryURL(), user, name)
			if version != "" {
				url += "?version=" + version
			}

			resp, err := http.Get(url) //nolint:gosec
			if err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("pull failed (%d): %s", resp.StatusCode, string(b))
			}

			content, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if pullOutput != "" {
				if err := os.WriteFile(pullOutput, content, 0o644); err != nil {
					return fmt.Errorf("failed to write file: %w", err)
				}
				fmt.Printf("SKILL.md saved to %s\n", pullOutput)
			} else {
				fmt.Print(string(content))
			}
			return nil
		},
	}
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Save to file instead of printing")

	// ── skill info <user/name> ─────────────────────────────────────────────
	infoCmd := &cobra.Command{
		Use:   "info <user/name>",
		Short: "Show detailed information about a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid skill reference, expected user/name")
			}

			client := registry.NewClient(getRegistryURL(), getToken())
			meta, err := client.Info(parts[0], parts[1])
			if err != nil {
				return err
			}

			fmt.Printf("Name:          %s/%s\n", meta.User, meta.Name)
			fmt.Printf("Version:       %s\n", meta.Version)
			fmt.Printf("Description:   %s\n", meta.Description)
			fmt.Printf("License:       %s\n", meta.License)
			fmt.Printf("Compatibility: %s\n", meta.Compatibility)
			fmt.Printf("Score:         %d/100\n", meta.Score)
			fmt.Printf("Published:     %s\n", meta.PublishedAt)
			return nil
		},
	}

	// ── skill login ────────────────────────────────────────────────────────
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Save your SkillHub API token locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Paste your SkillHub API token: ")
			var token string
			fmt.Scanln(&token)
			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(home, ".skillhub")
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
			tokenPath := filepath.Join(dir, "token")
			if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
				return err
			}
			fmt.Println("✓ Token saved to", tokenPath)
			return nil
		},
	}

	// ── skill update <dir> ────────────────────────────────────────────────
	updateCmd := &cobra.Command{
		Use:   "update <skill-dir>",
		Short: "Bump the patch version and re-publish",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillDir := args[0]

			skill, err := format.ParseSkillDir(skillDir)
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}

			currentVersion := skill.Frontmatter.Metadata["version"]
			if currentVersion == "" {
				currentVersion = "1.0.0"
			}

			newVersion, err := bumpPatch(currentVersion)
			if err != nil {
				return fmt.Errorf("version bump failed: %w", err)
			}
			fmt.Printf("Bumping version: %s → %s\n", currentVersion, newVersion)

			skillMdPath := filepath.Join(skillDir, "SKILL.md")
			content, err := os.ReadFile(skillMdPath)
			if err != nil {
				return err
			}
			updated := strings.ReplaceAll(string(content),
				fmt.Sprintf(`  version: "%s"`, currentVersion),
				fmt.Sprintf(`  version: "%s"`, newVersion))
			// Also try without quotes.
			updated = strings.ReplaceAll(updated,
				fmt.Sprintf("  version: %s", currentVersion),
				fmt.Sprintf("  version: %s", newVersion))
			if err := os.WriteFile(skillMdPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("failed to write SKILL.md: %w", err)
			}
			fmt.Printf("Updated SKILL.md to version %s\n", newVersion)

			skill, err = format.ParseSkillDir(skillDir)
			if err != nil {
				return err
			}
			result := format.Validate(skill)
			if !result.Valid {
				fmt.Println("Validation failed after version bump:")
				for _, e := range result.Errors {
					fmt.Printf("  ✗ %s\n", e)
				}
				os.Exit(1)
			}

			user := os.Getenv("SKILLHUB_USER")
			if user == "" {
				return fmt.Errorf("SKILLHUB_USER env var is required")
			}

			archivePath := filepath.Join(os.TempDir(), skill.Frontmatter.Name+".skill")
			if err := packer.Pack(skillDir, archivePath); err != nil {
				return fmt.Errorf("pack failed: %w", err)
			}
			defer os.Remove(archivePath)

			fmt.Printf("Publishing %s/%s@%s...\n", user, skill.Frontmatter.Name, newVersion)

			client := registry.NewClient(getRegistryURL(), getToken())
			resp, err := client.Publish(user, skill.Frontmatter.Name, newVersion, archivePath,
				skill.Frontmatter.Description,
				skill.Frontmatter.License,
				skill.Frontmatter.Compatibility,
				skill.Frontmatter.Metadata,
				result.Score,
			)
			if err != nil {
				return fmt.Errorf("publish failed: %w", err)
			}
			fmt.Printf("✓ Published: %s/%s@%s (score: %d/100)\n",
				resp.Skill.User, resp.Skill.Name, resp.Skill.Version, result.Score)
			return nil
		},
	}

	// ── skill init [name] ─────────────────────────────────────────────────
	initCmd := &cobra.Command{
		Use:   "init [skill-name]",
		Short: "Scaffold a new skill directory interactively",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				name = prompt(reader, "Skill name (lowercase, hyphens only): ")
			}
			name = strings.TrimSpace(strings.ToLower(name))
			if !regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`).MatchString(name) {
				return fmt.Errorf("invalid skill name '%s': use lowercase letters, digits, hyphens only", name)
			}

			description := prompt(reader, "Description (start with a verb): ")
			description = strings.TrimSpace(description)
			if description == "" {
				description = "Describe what this skill does and when to use it."
			}

			fmt.Println("License options: MIT, Apache-2.0, GPL-3.0, BSD-2-Clause, or leave blank")
			license := strings.TrimSpace(prompt(reader, "License [MIT]: "))
			if license == "" {
				license = "MIT"
			}

			fmt.Println("Model compatibility examples: 'Claude Code', 'Claude Sonnet 4', or leave blank")
			compatibility := strings.TrimSpace(prompt(reader, "Compatibility: "))

			author := os.Getenv("SKILLHUB_USER")
			if author == "" {
				author = strings.TrimSpace(prompt(reader, "Author (your username): "))
			}

			addScripts := promptBool(reader, "Add scripts/ directory? [y/N]: ")
			addRefs := promptBool(reader, "Add references/ directory? [y/N]: ")
			addAssets := promptBool(reader, "Add assets/ directory? [y/N]: ")

			if err := os.MkdirAll(name, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			today := time.Now().Format("2006-01-02")
			skillMd := fmt.Sprintf("---\nname: \"%s\"\ndescription: \"%s\"\nlicense: \"%s\"\ncompatibility: \"%s\"\nmetadata:\n  author: \"%s\"\n  version: \"1.0.0\"\n  created: \"%s\"\n  updated: \"%s\"\n---\n\n# %s\n\n## Overview\n<!-- Describe what this skill does in 2-3 sentences. -->\n\n## Prerequisites Checklist\n- [ ] <!-- Add prerequisite 1 -->\n\n## Step-by-Step Guide\n\n### 1. <!-- First step title -->\n<!-- Describe step 1 in detail. -->\n\n## Edge Cases & Troubleshooting\n<!-- Describe common errors and how to handle them. -->\n\n## Examples\n<!-- Provide 1-2 concrete examples of using this skill. -->\n",
				name, description, license, compatibility, author, today, today, name)

			if err := os.WriteFile(filepath.Join(name, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
				return fmt.Errorf("failed to write SKILL.md: %w", err)
			}

			writeDir := func(dir, content string) {
				_ = os.MkdirAll(filepath.Join(name, dir), 0o755)
				_ = os.WriteFile(filepath.Join(name, dir, "README.md"), []byte(content), 0o644)
			}
			if addScripts {
				writeDir("scripts", "# Scripts\n\nAdd helper scripts for this skill here.\n")
			}
			if addRefs {
				writeDir("references", "# References\n\nAdd reference documentation here.\n")
			}
			if addAssets {
				writeDir("assets", "# Assets\n\nAdd templates, configs, or other assets here.\n")
			}

			parsedSkill, err := format.ParseSkillDir(name)
			if err != nil {
				return fmt.Errorf("scaffold validation error: %w", err)
			}
			result := format.Validate(parsedSkill)

			fmt.Printf("\n✓ Skill scaffolded at ./%s/\n", name)
			fmt.Printf("  Score: %d/100\n", result.Score)
			if len(result.Warnings) > 0 {
				fmt.Println("  Warnings (fill in the template to fix these):")
				for _, w := range result.Warnings {
					fmt.Printf("    ! %s\n", w)
				}
			}
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  1. Edit ./%s/SKILL.md\n", name)
			fmt.Printf("  2. skill validate ./%s\n", name)
			fmt.Printf("  3. skill push ./%s\n", name)
			return nil
		},
	}

	rootCmd.AddCommand(loginCmd, initCmd, validateCmd, pushCmd, pullCmd, installCmd, searchCmd, infoCmd, updateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

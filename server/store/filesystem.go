package store

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/skillhub/skillhub/internal/registry"
)

type FilesystemStore struct {
	BaseDir string
}

func NewFilesystemStore(baseDir string) (*FilesystemStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FilesystemStore{BaseDir: baseDir}, nil
}

func (s *FilesystemStore) skillDir(user, name, version string) string {
	return filepath.Join(s.BaseDir, user, name, version)
}

// Save stores a skill archive and its metadata.
func (s *FilesystemStore) Save(meta registry.SkillMeta, archiveData []byte) error {
	dir := s.skillDir(meta.User, meta.Name, meta.Version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	archivePath := filepath.Join(dir, meta.Name+".skill")
	if err := os.WriteFile(archivePath, archiveData, 0644); err != nil {
		return fmt.Errorf("failed to write skill archive: %w", err)
	}

	meta.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	meta.DownloadURL = fmt.Sprintf("/api/v1/skills/%s/%s/download?version=%s",
		meta.User, meta.Name, meta.Version)

	metaPath := filepath.Join(dir, "meta.json")
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, metaBytes, 0644)
}

// GetMeta returns metadata for the latest version of a skill.
func (s *FilesystemStore) GetMeta(user, name string) (*registry.SkillMeta, error) {
	version, err := s.latestVersion(user, name)
	if err != nil {
		return nil, err
	}
	return s.GetMetaVersion(user, name, version)
}

// GetMetaVersion returns metadata for a specific version, with live download count.
func (s *FilesystemStore) GetMetaVersion(user, name, version string) (*registry.SkillMeta, error) {
	metaPath := filepath.Join(s.skillDir(user, name, version), "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("skill %s/%s@%s not found", user, name, version)
	}
	var meta registry.SkillMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	meta.Downloads = s.readDownloads(user, name)
	return &meta, nil
}

// GetArchive returns the raw .skill archive bytes.
func (s *FilesystemStore) GetArchive(user, name, version string) ([]byte, error) {
	if version == "" || version == "latest" {
		var err error
		version, err = s.latestVersion(user, name)
		if err != nil {
			return nil, err
		}
	}
	archivePath := filepath.Join(s.skillDir(user, name, version), name+".skill")
	return os.ReadFile(archivePath)
}

// Search returns skills matching the query string (searches name + description).
func (s *FilesystemStore) Search(query string) ([]registry.SkillMeta, error) {
	var results []registry.SkillMeta
	query = strings.ToLower(query)

	userDirs, err := os.ReadDir(s.BaseDir)
	if err != nil {
		return nil, err
	}

	for _, userDir := range userDirs {
		if !userDir.IsDir() || strings.HasPrefix(userDir.Name(), ".") {
			continue
		}
		skillDirs, err := os.ReadDir(filepath.Join(s.BaseDir, userDir.Name()))
		if err != nil {
			continue
		}
		for _, skillDir := range skillDirs {
			if !skillDir.IsDir() {
				continue
			}
			meta, err := s.GetMeta(userDir.Name(), skillDir.Name())
			if err != nil {
				continue
			}
			if query == "" ||
				strings.Contains(strings.ToLower(meta.Name), query) ||
				strings.Contains(strings.ToLower(meta.Description), query) {
				results = append(results, *meta)
			}
		}
	}

	return results, nil
}

// Stats returns aggregate counts across the whole registry.
func (s *FilesystemStore) Stats() Stats {
	var stats Stats

	userDirs, err := os.ReadDir(s.BaseDir)
	if err != nil {
		return stats
	}

	for _, userDir := range userDirs {
		if !userDir.IsDir() || strings.HasPrefix(userDir.Name(), ".") {
			continue
		}
		stats.TotalCreators++
		skillDirs, _ := os.ReadDir(filepath.Join(s.BaseDir, userDir.Name()))
		for _, skillDir := range skillDirs {
			if !skillDir.IsDir() {
				continue
			}
			stats.TotalSkills++
			stats.TotalDownloads += s.readDownloads(userDir.Name(), skillDir.Name())
		}
	}

	return stats
}

// IncrementDownloads adds 1 to the download counter for a skill.
func (s *FilesystemStore) IncrementDownloads(user, name string) {
	count := s.readDownloads(user, name)
	count++
	path := filepath.Join(s.BaseDir, user, name, ".downloads")
	os.WriteFile(path, []byte(strconv.Itoa(count)), 0644) //nolint:errcheck
}

func (s *FilesystemStore) readDownloads(user, name string) int {
	path := filepath.Join(s.BaseDir, user, name, ".downloads")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

// CreateUserToken generates a new random token for the given username.
// Returns an error if the user already has a token.
func (s *FilesystemStore) CreateUserToken(username string) (string, error) {
	dir := filepath.Join(s.BaseDir, ".tokens")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, username)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("user %q already has a token; use your existing token or contact the admin", username)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	return token, os.WriteFile(path, []byte(token), 0600)
}

// ValidateUserToken checks whether the given token matches the stored token for username.
func (s *FilesystemStore) ValidateUserToken(username, token string) bool {
	path := filepath.Join(s.BaseDir, ".tokens", username)
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == token
}

// GetSkillMdBody reads the Markdown body from the SKILL.md inside the stored archive.
// It extracts the archive into a temp directory, reads SKILL.md, and returns the body text.
func (s *FilesystemStore) GetSkillMdBody(user, name, version string) (string, error) {
	archivePath := filepath.Join(s.skillDir(user, name, version), name+".skill")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "skillhub-body-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// Write archive to temp file then unpack
	tmpArchive := filepath.Join(tmpDir, name+".skill")
	if err := os.WriteFile(tmpArchive, data, 0644); err != nil {
		return "", err
	}

	if err := unpackForRead(tmpArchive, tmpDir); err != nil {
		return "", err
	}

	skillMdPath := filepath.Join(tmpDir, name, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", err
	}

	// Strip frontmatter (---\n...\n---\n)
	s2 := string(content)
	if len(s2) > 3 && s2[:3] == "---" {
		rest := s2[3:]
		if len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
		idx := strings.Index(rest, "\n---")
		if idx >= 0 {
			body := rest[idx+4:]
			if len(body) > 0 && body[0] == '\n' {
				body = body[1:]
			}
			return body, nil
		}
	}
	return s2, nil
}

func (s *FilesystemStore) latestVersion(user, name string) (string, error) {
	skillBase := filepath.Join(s.BaseDir, user, name)
	versions, err := os.ReadDir(skillBase)
	if err != nil {
		return "", fmt.Errorf("skill %s/%s not found", user, name)
	}
	var vnames []string
	for _, v := range versions {
		if v.IsDir() {
			vnames = append(vnames, v.Name())
		}
	}
	if len(vnames) == 0 {
		return "", fmt.Errorf("no versions found for %s/%s", user, name)
	}
	sort.Strings(vnames)
	return vnames[len(vnames)-1], nil
}

// unpackForRead extracts a .skill tar.gz archive into destDir (read-only helper, no path-traversal guard needed for local files).
func unpackForRead(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.Contains(hdr.Name, "..") {
			continue
		}
		target := filepath.Join(destDir, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755) //nolint:errcheck
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755) //nolint:errcheck
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			io.Copy(out, tr) //nolint:errcheck
			out.Close()
		}
	}
	return nil
}

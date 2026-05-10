package registry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	BaseURL string
	Token   string
}

func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token}
}

// Publish sends a .skill archive to the registry.
func (c *Client) Publish(user, name, version, archivePath, description, license, compatibility string, metadata map[string]string, score int) (*PublishResponse, error) {
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	req := PublishRequest{
		User:          user,
		Name:          name,
		Version:       version,
		Description:   description,
		License:       license,
		Compatibility: compatibility,
		Metadata:      metadata,
		Score:         score,
		Archive:       base64.StdEncoding.EncodeToString(data),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/skillhub/v1/skills/%s/%s", c.BaseURL, user, name),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("publish failed (%d): %s", resp.StatusCode, string(b))
	}

	var result PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Download fetches a .skill archive and saves it to destPath.
func (c *Client) Download(user, name, version, destPath string) error {
	url := fmt.Sprintf("%s/api/skillhub/v1/skills/%s/%s/download", c.BaseURL, user, name)
	if version != "" {
		url += "?version=" + version
	}

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(b))
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// Search queries the registry.
func (c *Client) Search(query string) (*SearchResult, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/skillhub/v1/search?q=%s", c.BaseURL, query)) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Info fetches skill metadata.
func (c *Client) Info(user, name string) (*SkillMeta, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/skillhub/v1/skills/%s/%s", c.BaseURL, user, name)) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skill %s/%s not found", user, name)
	}

	var meta SkillMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

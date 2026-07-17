package ghcomment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Token   string
	APIBase string
	Repo    string
	Client  *http.Client
}

func ConfigFromEnv() (Config, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN is not set -- pass it via `env: GITHUB_TOKEN: ${{ github.token }}` in the workflow")
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return Config{}, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}
	apiBase := os.Getenv("GITHUB_API_URL")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	return Config{
		Token:   token,
		APIBase: apiBase,
		Repo:    repo,
		Client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type ghComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

func PostOrUpdate(cfg Config, prNumber int, body string) error {
	existingID, err := findExistingComment(cfg, prNumber)
	if err != nil {
		return fmt.Errorf("listing existing comments: %w", err)
	}

	if existingID != 0 {
		url := fmt.Sprintf("%s/repos/%s/issues/comments/%d", cfg.APIBase, cfg.Repo, existingID)
		return doRequest(cfg, http.MethodPatch, url, ghComment{Body: body}, nil)
	}

	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", cfg.APIBase, cfg.Repo, prNumber)
	return doRequest(cfg, http.MethodPost, url, ghComment{Body: body}, nil)
}

func findExistingComment(cfg Config, prNumber int) (int64, error) {
	page := 1
	for {
		url := fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100&page=%d", cfg.APIBase, cfg.Repo, prNumber, page)
		var comments []ghComment
		if err := doRequest(cfg, http.MethodGet, url, nil, &comments); err != nil {
			return 0, err
		}
		for _, c := range comments {
			if strings.Contains(c.Body, Marker) {
				return c.ID, nil
			}
		}
		if len(comments) < 100 {
			return 0, nil
		}
		page++
	}
}

func doRequest(cfg Config, method, url string, reqBody interface{}, respOut interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s -> %d: %s", method, url, resp.StatusCode, string(data))
	}
	if respOut != nil {
		if err := json.Unmarshal(data, respOut); err != nil {
			return fmt.Errorf("decoding response from %s: %w", url, err)
		}
	}
	return nil
}
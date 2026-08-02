package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"vyala/internal/findings"
)

// Payload matches the ScanRequest struct from the backend
type Payload struct {
	RepoFullName string        `json:"repo_full_name"`
	GithubRepoID int64         `json:"github_repo_id"`
	CommitSHA    string        `json:"commit_sha"`
	Branch       string        `json:"branch"`
	TriggerType  string        `json:"trigger_type"`
	IsPrivate    bool          `json:"is_private"`
	Status       string        `json:"status"`
	CBOM         findings.CBOM `json:"cbom"`
}

func buildPayload(cbom findings.CBOM, repoFullName, commitSHA, branch, triggerType string, githubRepoID int64, isPrivate bool) Payload {
	return Payload{
		RepoFullName: repoFullName,
		GithubRepoID: githubRepoID,
		CommitSHA:    commitSHA,
		Branch:       branch,
		TriggerType:  triggerType,
		IsPrivate:    isPrivate,
		Status:       "success",
		CBOM:         cbom,
	}
}

func UploadCBOM(cbom findings.CBOM, apiURL, apiKey string) error {
	// Gather metadata from GitHub Actions environment variables
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	commitSHA := os.Getenv("GITHUB_SHA")
	branch := os.Getenv("GITHUB_REF")
	triggerType := os.Getenv("GITHUB_EVENT_NAME")

	// GITHUB_REF looks like "refs/heads/main", we just want "main"
	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	// For local testing, provide fallbacks
	if repoFullName == "" {
		repoFullName = "local-test/repo"
	}
	if commitSHA == "" {
		commitSHA = "local-sha"
	}
	if branch == "" {
		branch = "main"
	}
	if triggerType == "" {
		triggerType = "manual"
	}

	// Generate a dummy GithubRepoID for local testing if not in CI
	var githubRepoID int64 = 1
	if os.Getenv("GITHUB_REPOSITORY_ID") != "" {
		// In real CI, this env var is available
		// (We'll use a real parsing function later, for now just fallback to 1 if not present)
		githubRepoID = 1
	}

	payload := buildPayload(cbom, repoFullName, commitSHA, branch, triggerType, githubRepoID, false)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading CBOM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("backend rejected upload with status: %s", resp.Status)
	}

	fmt.Println("✅ CBOM successfully uploaded to VYALA backend.")
	return nil
}

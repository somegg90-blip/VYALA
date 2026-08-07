package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"vyala/internal/findings"
)

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
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	commitSHA := os.Getenv("GITHUB_SHA")
	branch := os.Getenv("GITHUB_REF")
	triggerType := os.Getenv("GITHUB_EVENT_NAME")

	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	if repoFullName == "" { repoFullName = "local-test/repo" }
	if commitSHA == "" { commitSHA = "local-sha" }
	if branch == "" { branch = "main" }
	if triggerType == "" { triggerType = "manual" }

	var githubRepoID int64 = 1
	if repoIDStr := os.Getenv("GITHUB_REPOSITORY_ID"); repoIDStr != "" {
		if parsed, err := strconv.ParseInt(repoIDStr, 10, 64); err == nil {
			githubRepoID = parsed
		}
	}

	payload := buildPayload(cbom, repoFullName, commitSHA, branch, triggerType, githubRepoID, false)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	// FIX: Add strict timeout to prevent CI hangs
	client := &http.Client{Timeout: 30 * time.Second}
	
	var resp *http.Response
	
	// FIX: Retry loop for network resilience
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err = client.Do(req)
		
		// Success condition: 200 OK or 201 Created
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
			resp.Body.Close()
			fmt.Println("✅ CBOM successfully uploaded to VYALA backend.")
			return nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("uploading CBOM failed after 3 attempts: %w", err)
	}
	return fmt.Errorf("backend rejected upload after 3 attempts with status: %s", resp.Status)
}
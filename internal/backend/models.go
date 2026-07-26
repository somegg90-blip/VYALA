package backend

import (
	"time"

	"github.com/google/uuid"
)

var AccountIDPlaceholder = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type ScanRequest struct {
	RepoFullName string `json:"repo_full_name"`
	GithubRepoID int64  `json:"github_repo_id"`
	CommitSHA    string `json:"commit_sha"`
	Branch       string `json:"branch"`
	TriggerType  string `json:"trigger_type"`
	IsPrivate    bool   `json:"is_private"`
	CBOM         struct {
		Findings []FindingInput `json:"findings"`
	} `json:"cbom"`
}

type FindingInput struct {
	ID                   string `json:"finding_id"`
	Type                 string `json:"type"`
	File                 string `json:"file"`
	Line                 int    `json:"line"`
	Algorithm            string `json:"algorithm"`
	Severity             string `json:"severity"`
	Category             string `json:"category"`
	ExposureEstimate     string `json:"hnd_exposure_estimate"`
	SuggestedReplacement string `json:"suggested_replacement"`
	RuleID               string `json:"rule_id"`
}

// DB Models
type Repository struct {
	ID           uuid.UUID `json:"id"`
	AccountID    uuid.UUID `json:"account_id"`
	GithubRepoID int64     `json:"github_repo_id"`
	FullName     string    `json:"full_name"`
	IsPrivate    bool      `json:"is_private"`
	CreatedAt    time.Time `json:"created_at"`
}

type Scan struct {
	ID          uuid.UUID `json:"id"`
	RepoID      uuid.UUID `json:"repo_id"`
	CommitSHA   string    `json:"commit_sha"`
	Branch      string    `json:"branch"`
	TriggerType string    `json:"trigger_type"`
	CreatedAt   time.Time `json:"created_at"`
}

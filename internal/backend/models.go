package backend

import (
	"time"
	"vyala/internal/findings"

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
	Status       string `json:"status"`
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

type RepositorySummary struct {
	ID            uuid.UUID `json:"id"`
	FullName      string    `json:"full_name"`
	OpenFindings  int       `json:"open_findings"`
	HighSeverity  int       `json:"high_severity"`
	LastScannedAt time.Time `json:"last_scanned_at"`
}

type TrendPoint struct {
	Date   string `json:"date"`
	High   int64  `json:"high"`
	Medium int64  `json:"medium"`
	Low    int64  `json:"low"`
}

type Finding struct {
	ID                   uuid.UUID `json:"id"`
	RepoID               uuid.UUID `json:"repo_id"`
	FindingID            string    `json:"finding_id"`
	Type                 string    `json:"type"`
	File                 string    `json:"file"`
	Line                 int       `json:"line"`
	Algorithm            string    `json:"algorithm"`
	Severity             string    `json:"severity"`
	Category             string    `json:"category"`
	ExposureEstimate     string    `json:"hnd_exposure_estimate"`
	SuggestedReplacement string    `json:"suggested_replacement"`
	RuleID               string    `json:"rule_id"`
	Status               string    `json:"status"`
}

// Reuse the existing findings schema for CBOM payloads.
type ScanResult struct {
	Findings []findings.Finding `json:"findings"`
}

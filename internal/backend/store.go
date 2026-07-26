package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func (s *Store) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}

func NewStore(dbURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &Store{db: pool}, nil
}

// IngestScan handles the complex logic of saving a scan and updating finding lifecycles.
func (s *Store) IngestScan(ctx context.Context, req ScanRequest, accountID uuid.UUID) (uuid.UUID, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx) // Safe to call after commit

	// 1. Upsert Repository
	var repoID uuid.UUID
	err = tx.QueryRow(ctx, `
        INSERT INTO repositories (account_id, github_repo_id, full_name, is_private)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (github_repo_id) DO UPDATE SET full_name = EXCLUDED.full_name
        RETURNING id
    `, accountID, req.GithubRepoID, req.RepoFullName, req.IsPrivate).Scan(&repoID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to upsert repository: %w", err)
	}

	// 2. Create Scan Record
	rawCBOM, _ := json.Marshal(req.CBOM)
	var scanID uuid.UUID
	err = tx.QueryRow(ctx, `
        INSERT INTO scans (repo_id, commit_sha, branch, trigger_type, raw_cbom)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, repoID, req.CommitSHA, req.Branch, req.TriggerType, rawCBOM).Scan(&scanID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert scan: %w", err)
	}

	// 3. Process Findings (Upsert + Lifecycle)
	seenFindingIDs := make([]string, 0, len(req.CBOM.Findings))
	seenFindingSet := make(map[string]struct{}, len(req.CBOM.Findings))

	for _, f := range req.CBOM.Findings {
		if _, exists := seenFindingSet[f.ID]; exists {
			continue
		}
		seenFindingSet[f.ID] = struct{}{}
		seenFindingIDs = append(seenFindingIDs, f.ID)

		// Upsert finding: if it exists and is open, just update last_seen.
		// If it exists but was resolved (regression), reopen it.
		// If it doesn't exist, create it.
		_, err = tx.Exec(ctx, `
            INSERT INTO findings (
                repo_id, finding_id, type, file, line, algorithm, severity, 
                category, exposure_estimate, suggested_replacement, rule_id, 
                status, first_seen_scan_id, last_seen_scan_id, first_seen_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'open', $12, $12, NOW())
            ON CONFLICT (repo_id, finding_id) DO UPDATE SET 
                last_seen_scan_id = EXCLUDED.last_seen_scan_id,
                status = 'open',
                resolved_at = NULL
        `,
			repoID, f.ID, f.Type, f.File, f.Line, f.Algorithm, f.Severity,
			f.Category, f.ExposureEstimate, f.SuggestedReplacement, f.RuleID,
			scanID,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to upsert finding %s: %w", f.ID, err)
		}
	}

	// 4. Mark missing findings as resolved
	// Any finding for this repo that is currently 'open' but NOT in the seenFindingIDs map
	// gets marked as resolved.
	if len(seenFindingIDs) > 0 {
		// Build a parameterized IN clause
		args := []interface{}{repoID}
		query := `
            UPDATE findings 
            SET status = 'resolved', resolved_at = NOW() 
            WHERE repo_id = $1 
              AND status = 'open' 
              AND finding_id NOT IN (
        `
		for i, id := range seenFindingIDs {
			if i > 0 {
				query += ","
			}
			query += fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}
		query += ")"

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resolve old findings: %w", err)
		}
	} else {
		// If the scan had zero findings, resolve all open findings for this repo
		_, err = tx.Exec(ctx, `
            UPDATE findings 
            SET status = 'resolved', resolved_at = NOW() 
            WHERE repo_id = $1 AND status = 'open'
        `, repoID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to resolve all old findings: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	return scanID, nil
}

// RepositorySummary is used for the dashboard list view
type RepositorySummary struct {
	ID            uuid.UUID `json:"id"`
	FullName      string    `json:"full_name"`
	OpenFindings  int       `json:"open_findings"`
	HighSeverity  int       `json:"high_severity"`
	LastScannedAt time.Time `json:"last_scanned_at"`
}

func (s *Store) GetRepos(ctx context.Context, accountID uuid.UUID) ([]RepositorySummary, error) {
	rows, err := s.db.Query(ctx, `
        SELECT 
            r.id, r.full_name, 
            COUNT(f.id) FILTER (WHERE f.status = 'open') as open_findings,
            COUNT(f.id) FILTER (WHERE f.status = 'open' AND f.severity = 'high') as high_severity,
            (SELECT MAX(created_at) FROM scans WHERE repo_id = r.id) as last_scanned_at
        FROM repositories r
        LEFT JOIN findings f ON r.id = f.repo_id
        WHERE r.account_id = $1
        GROUP BY r.id, r.full_name
        ORDER BY last_scanned_at DESC NULLS LAST
    `, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []RepositorySummary
	for rows.Next() {
		var r RepositorySummary
		if err := rows.Scan(&r.ID, &r.FullName, &r.OpenFindings, &r.HighSeverity, &r.LastScannedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, nil
}

// TrendPoint represents a single day's posture
type TrendPoint struct {
	Date   string `json:"date"`
	High   int    `json:"high"`
	Medium int    `json:"medium"`
	Low    int    `json:"low"`
}

func (s *Store) GetTrends(ctx context.Context, repoID uuid.UUID, days int) ([]TrendPoint, error) {
	// generate_series creates a row for every day in the past X days.
	// We then count findings that were open on that specific day.
	query := fmt.Sprintf(`
        WITH days AS (
            SELECT generate_series(
                date_trunc('day', NOW() - INTERVAL '%d days'),
                date_trunc('day', NOW()),
                INTERVAL '1 day'
            ) AS day
        )
        SELECT
            to_char(d.day, 'YYYY-MM-DD') as date,
            COUNT(f.id) FILTER (WHERE f.severity = 'high' AND f.first_seen_at < (d.day + INTERVAL '1 day') AND (f.resolved_at IS NULL OR f.resolved_at > d.day)) AS high,
            COUNT(f.id) FILTER (WHERE f.severity = 'medium' AND f.first_seen_at < (d.day + INTERVAL '1 day') AND (f.resolved_at IS NULL OR f.resolved_at > d.day)) AS medium,
            COUNT(f.id) FILTER (WHERE f.severity = 'low' AND f.first_seen_at < (d.day + INTERVAL '1 day') AND (f.resolved_at IS NULL OR f.resolved_at > d.day)) AS low
        FROM days d
        LEFT JOIN findings f ON f.repo_id = $1
        GROUP BY d.day
        ORDER BY d.day ASC
    `, days)

	rows, err := s.db.Query(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []TrendPoint
	for rows.Next() {
		var t TrendPoint
		if err := rows.Scan(&t.Date, &t.High, &t.Medium, &t.Low); err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}
	return trends, nil
}

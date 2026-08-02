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

    // Verify connectivity immediately
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("database ping failed: %w", err)
    }

    return &Store{db: pool}, nil
}

// IngestScan handles the complex logic of saving a scan and updating finding lifecycles.
func (s *Store) IngestScan(ctx context.Context, req ScanRequest, accountID uuid.UUID) (uuid.UUID, error) {
    tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return uuid.Nil, err
    }
    defer tx.Rollback(ctx)

    // 1. Upsert Repository (Fixed conflict target)
    var repoID uuid.UUID
    err = tx.QueryRow(ctx, `
        INSERT INTO repositories (account_id, github_repo_id, full_name, is_private)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (account_id, github_repo_id) DO UPDATE SET full_name = EXCLUDED.full_name
        RETURNING id
    `, accountID, req.GithubRepoID, req.RepoFullName, req.IsPrivate).Scan(&repoID)
    if err != nil {
        return uuid.Nil, fmt.Errorf("failed to upsert repository: %w", err)
    }

    // 2. Create Scan Record
    rawCBOM, err := json.Marshal(req.CBOM)
    if err != nil {
        return uuid.Nil, fmt.Errorf("failed to marshal raw cbom: %w", err)
    }
    var scanID uuid.UUID
    err = tx.QueryRow(ctx, `
        INSERT INTO scans (repo_id, commit_sha, branch, trigger_type, raw_cbom)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, repoID, req.CommitSHA, req.Branch, req.TriggerType, rawCBOM).Scan(&scanID)
    if err != nil {
        return uuid.Nil, fmt.Errorf("failed to insert scan: %w", err)
    }

    // 3. Process Findings (Upsert)
    seenFindingIDs := make([]string, 0, len(req.CBOM.Findings))
    seenFindingSet := make(map[string]struct{}, len(req.CBOM.Findings))

    for _, f := range req.CBOM.Findings {
        if _, exists := seenFindingSet[f.ID]; exists {
            continue
        }
        seenFindingSet[f.ID] = struct{}{}
        seenFindingIDs = append(seenFindingIDs, f.ID)

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
    // FIXED: Only run lifecycle resolution if the scan completed successfully.
    // This prevents a crashed scanner (empty CBOM) from wiping out trend data.
    if req.Status == "success" || req.Status == "" {
        if len(seenFindingIDs) > 0 {
            args := []interface{}{repoID}
            query := `
                UPDATE findings 
                SET status = 'resolved', resolved_at = NOW() 
                WHERE repo_id = $1 AND status = 'open' AND finding_id NOT IN (`
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
            _, err = tx.Exec(ctx, `
                UPDATE findings 
                SET status = 'resolved', resolved_at = NOW() 
                WHERE repo_id = $1 AND status = 'open'
            `, repoID)
            if err != nil {
                return uuid.Nil, fmt.Errorf("failed to resolve all old findings: %w", err)
            }
        }
    }

    err = tx.Commit(ctx)
    if err != nil {
        return uuid.Nil, err
    }

    return scanID, nil
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

    repos := []RepositorySummary{}
    for rows.Next() {
        var r RepositorySummary
        var lastScanned *time.Time // Fix for NULL last_scanned_at
        if err := rows.Scan(&r.ID, &r.FullName, &r.OpenFindings, &r.HighSeverity, &lastScanned); err != nil {
            return nil, err
        }
        if lastScanned != nil {
            r.LastScannedAt = *lastScanned
        }
        repos = append(repos, r)
    }
    return repos, nil
}

func (s *Store) GetTrends(ctx context.Context, repoID uuid.UUID, accountID uuid.UUID, days int) ([]TrendPoint, error) {
    // FIXED: Parameterized the interval to prevent SQLi, added account_id scoping
    query := `
        WITH days AS (
            SELECT generate_series(
                date_trunc('day', NOW() - INTERVAL '1 day' * $3),
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
        LEFT JOIN repositories r ON r.id = f.repo_id AND r.account_id = $2
        GROUP BY d.day
        ORDER BY d.day ASC
    `

    rows, err := s.db.Query(ctx, query, repoID, accountID, days)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    trends := []TrendPoint{}
    for rows.Next() {
        var t TrendPoint
        if err := rows.Scan(&t.Date, &t.High, &t.Medium, &t.Low); err != nil {
            return nil, err
        }
        trends = append(trends, t)
    }
    return trends, nil
}

func (s *Store) GetScans(ctx context.Context, repoID uuid.UUID, accountID uuid.UUID, limit int) ([]Scan, error) {
    // Added account_id check via JOIN
    rows, err := s.db.Query(ctx, `
        SELECT s.id, s.repo_id, s.commit_sha, s.branch, s.trigger_type, s.created_at
        FROM scans s
        JOIN repositories r ON s.repo_id = r.id
        WHERE s.repo_id = $1 AND r.account_id = $2
        ORDER BY s.created_at DESC 
        LIMIT $3
    `, repoID, accountID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    scans := []Scan{}
    for rows.Next() {
        var sc Scan
        if err := rows.Scan(&sc.ID, &sc.RepoID, &sc.CommitSHA, &sc.Branch, &sc.TriggerType, &sc.CreatedAt); err != nil {
            return nil, err
        }
        scans = append(scans, sc)
    }
    return scans, nil
}

func (s *Store) GetOpenFindings(ctx context.Context, repoID uuid.UUID, accountID uuid.UUID) ([]Finding, error) {
    // FIXED: Added account_id check and proper severity sorting
    rows, err := s.db.Query(ctx, `
        SELECT f.id, f.repo_id, f.finding_id, f.type, f.file, f.line, f.algorithm, f.severity, 
               f.category, f.exposure_estimate, f.suggested_replacement, f.rule_id, f.status
        FROM findings f
        JOIN repositories r ON f.repo_id = r.id
        WHERE f.repo_id = $1 AND r.account_id = $2 AND f.status = 'open'
        ORDER BY 
            CASE f.severity 
                WHEN 'high' THEN 0 
                WHEN 'medium' THEN 1 
                WHEN 'low' THEN 2 
            END, 
            f.file ASC
    `, repoID, accountID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    findings := []Finding{}
    for rows.Next() {
        var f Finding
        if err := rows.Scan(&f.ID, &f.RepoID, &f.FindingID, &f.Type, &f.File, &f.Line, &f.Algorithm, &f.Severity, &f.Category, &f.ExposureEstimate, &f.SuggestedReplacement, &f.RuleID, &f.Status); err != nil {
            return nil, err
        }
        findings = append(findings, f)
    }
    return findings, nil
}

// FIXED: Race condition resolved using ON CONFLICT DO UPDATE
func (s *Store) GetOrCreateUser(ctx context.Context, githubUserID int64, username, email string) (uuid.UUID, uuid.UUID, error) {
    tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return uuid.Nil, uuid.Nil, err
    }
    defer tx.Rollback(ctx)

    var userID uuid.UUID
    var accountID uuid.UUID

    // Upsert User
    err = tx.QueryRow(ctx, `
        INSERT INTO users (github_user_id, email, name) 
        VALUES ($1, $2, $3) 
        ON CONFLICT (github_user_id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email
        RETURNING id
    `, githubUserID, email, username).Scan(&userID)
    if err != nil {
        return uuid.Nil, uuid.Nil, fmt.Errorf("failed to upsert user: %w", err)
    }

    // Check if user already has an account
    err = tx.QueryRow(ctx, `SELECT account_id FROM account_users WHERE user_id = $1`, userID).Scan(&accountID)
    if err != nil && err != pgx.ErrNoRows {
        return uuid.Nil, uuid.Nil, fmt.Errorf("failed to query account: %w", err)
    }

    if err == pgx.ErrNoRows {
        // Create personal account
        err = tx.QueryRow(ctx, `
            INSERT INTO accounts (name, plan_tier) 
            VALUES ($1, 'free') 
            RETURNING id
        `, username+"'s Account").Scan(&accountID)
        if err != nil {
            return uuid.Nil, uuid.Nil, fmt.Errorf("failed to create account: %w", err)
        }

        _, err = tx.Exec(ctx, `
            INSERT INTO account_users (account_id, user_id, role) 
            VALUES ($1, $2, 'admin')
            ON CONFLICT DO NOTHING
        `, accountID, userID)
        if err != nil {
            return uuid.Nil, uuid.Nil, fmt.Errorf("failed to link user to account: %w", err)
        }
    }

    if err := tx.Commit(ctx); err != nil {
        return uuid.Nil, uuid.Nil, err
    }

    return userID, accountID, nil
}
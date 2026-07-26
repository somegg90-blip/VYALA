package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"

    "vyala/internal/backend"

    "github.com/google/uuid"
    "github.com/joho/godotenv"
)

func main() {
    _ = godotenv.Load() // Ignore error if .env doesn't exist in prod

    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL must be set")
    }

    store, err := backend.NewStore(dbURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer store.Close()

    mux := http.NewServeMux()

    // POST /v1/scans - Ingest scan results
    mux.HandleFunc("POST /v1/scans", func(w http.ResponseWriter, r *http.Request) {
        // TODO: In production, validate API Key here and extract accountID
        // For now, we hardcode a dummy account ID for testing
        accountID := backend.AccountIDPlaceholder

        var req backend.ScanRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        // Basic validation
        if req.RepoFullName == "" || req.CommitSHA == "" || req.GithubRepoID == 0 {
            http.Error(w, "Missing required fields (repo_full_name, github_repo_id, commit_sha)", http.StatusBadRequest)
            return
        }

        scanID, err := store.IngestScan(context.Background(), req, accountID)
        if err != nil {
            log.Printf("Error ingesting scan: %v", err)
            http.Error(w, "Failed to ingest scan", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID.String()})
    })

    // GET /v1/repos - List repositories for the dashboard
    mux.HandleFunc("GET /v1/repos", func(w http.ResponseWriter, r *http.Request) {
        accountID := backend.AccountIDPlaceholder
        repos, err := store.GetRepos(context.Background(), accountID)
        if err != nil {
            log.Printf("Error fetching repos: %v", err)
            http.Error(w, "Failed to fetch repos", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(repos)
    })

    // GET /v1/repos/{repo_id}/trends - Get 90-day trend data
    mux.HandleFunc("GET /v1/repos/{repo_id}/trends", func(w http.ResponseWriter, r *http.Request) {
        repoIDStr := r.PathValue("repo_id")
        repoID, err := uuid.Parse(repoIDStr)
        if err != nil {
            http.Error(w, "Invalid repo ID", http.StatusBadRequest)
            return
        }

        trends, err := store.GetTrends(context.Background(), repoID, 90)
        if err != nil {
            log.Printf("Error fetching trends: %v", err)
            http.Error(w, "Failed to fetch trends", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(trends)
    })

    // Health check
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("VYALA backend listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
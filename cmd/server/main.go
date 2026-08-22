package main

import (
    "bytes"
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "vyala/internal/backend"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "github.com/joho/godotenv"
)

var (
    githubClientID     string
    githubClientSecret string
    oauthRedirectURL   string
    sessionSecret      string
    frontendURL        string
    cliAPIKey          string
    isProduction       bool
)

// Typed JWT Claims to prevent panics
type VyalaClaims struct {
    UserID    string `json:"user_id"`
    AccountID string `json:"account_id"`
    jwt.RegisteredClaims
}

func main() {
    _ = godotenv.Load()

    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL must be set")
    }

    githubClientID = os.Getenv("GITHUB_CLIENT_ID")
    githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
    oauthRedirectURL = os.Getenv("OAUTH_REDIRECT_URL")
    sessionSecret = os.Getenv("SESSION_SECRET")
    frontendURL = os.Getenv("FRONTEND_URL")
    cliAPIKey = os.Getenv("CLI_API_KEY")
    isProduction = os.Getenv("ENV") == "production"

    // Validate critical secrets
    if sessionSecret == "" || len(sessionSecret) < 32 {
        log.Fatal("SESSION_SECRET must be set and at least 32 characters long")
    }
    if frontendURL == "" {
        frontendURL = "http://localhost:5173"
    }

    store, err := backend.NewStore(dbURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer store.Close()

    mux := http.NewServeMux()

    // --- Auth Routes ---
    mux.HandleFunc("GET /auth/github", func(w http.ResponseWriter, r *http.Request) {
        // Generate secure state for CSRF protection
        stateBytes := make([]byte, 32)
        if _, err := rand.Read(stateBytes); err != nil {
            http.Error(w, "Failed to generate state", http.StatusInternalServerError)
            return
        }
        state := hex.EncodeToString(stateBytes)

        http.SetCookie(w, &http.Cookie{
            Name:     "oauth_state",
            Value:    state,
            Path:     "/",
            HttpOnly: true,
            Secure:   isProduction,
            MaxAge:   600, // 10 minutes
            SameSite: http.SameSiteLaxMode,
        })

        url := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s", githubClientID, oauthRedirectURL, state)
        http.Redirect(w, r, url, http.StatusTemporaryRedirect)
    })

    mux.HandleFunc("GET /auth/github/callback", func(w http.ResponseWriter, r *http.Request) {
        // Verify state
        stateCookie, err := r.Cookie("oauth_state")
        if err != nil || stateCookie.Value == "" {
            http.Error(w, "State cookie missing", http.StatusBadRequest)
            return
        }
        stateParam := r.URL.Query().Get("state")
        if stateParam == "" || stateParam != stateCookie.Value {
            http.Error(w, "State mismatch", http.StatusBadRequest)
            return
        }

        // Clear state cookie
        http.SetCookie(w, &http.Cookie{
            Name: "oauth_state", Value: "", Path: "/", MaxAge: -1,
        })

        code := r.URL.Query().Get("code")
        if code == "" {
            http.Error(w, "Code not found", http.StatusBadRequest)
            return
        }

        // Use a client with timeouts
        client := &http.Client{Timeout: 10 * time.Second}

        // Exchange code for access token. Send credentials in the POST body
        // (never the query string, which leaks into proxies and access logs).
        tokenPayload, _ := json.Marshal(map[string]string{
            "client_id":     githubClientID,
            "client_secret": githubClientSecret,
            "code":          code,
            "redirect_uri":  oauthRedirectURL,
        })
        tokenReq, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewReader(tokenPayload))
        if err != nil {
            http.Error(w, "Failed to build token request", http.StatusInternalServerError)
            return
        }
        tokenReq.Header.Set("Accept", "application/json")
        tokenReq.Header.Set("Content-Type", "application/json")

        resp, err := client.Do(tokenReq)
        if err != nil {
            http.Error(w, "Failed to get token", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        var tokenResp struct {
            AccessToken string `json:"access_token"`
        }
        json.NewDecoder(resp.Body).Decode(&tokenResp)

        if tokenResp.AccessToken == "" {
            http.Error(w, "Failed to get access token", http.StatusUnauthorized)
            return
        }

        // Fetch user profile
        req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
        req.Header.Set("Authorization", "token "+tokenResp.AccessToken)
        ghResp, err := client.Do(req)
        if err != nil {
            http.Error(w, "Failed to fetch GitHub user", http.StatusInternalServerError)
            return
        }
        defer ghResp.Body.Close()

        if ghResp.StatusCode != http.StatusOK {
            http.Error(w, "GitHub API error", http.StatusUnauthorized)
            return
        }

        var ghUser struct {
            ID    int64  `json:"id"`
            Login string `json:"login"`
            Email string `json:"email"`
        }
        body, _ := io.ReadAll(ghResp.Body)
        json.Unmarshal(body, &ghUser)

        if ghUser.ID == 0 {
            http.Error(w, "Invalid GitHub user", http.StatusUnauthorized)
            return
        }

        if ghUser.Email == "" {
            emailReq, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
            emailReq.Header.Set("Authorization", "token "+tokenResp.AccessToken)
            emailResp, err := client.Do(emailReq)
            if err == nil {
                defer emailResp.Body.Close()
                var emails []struct {
                    Email   string `json:"email"`
                    Primary bool   `json:"primary"`
                }
                json.NewDecoder(emailResp.Body).Decode(&emails)
                for _, e := range emails {
                    if e.Primary {
                        ghUser.Email = e.Email
                        break
                    }
                }
            }
        }

        userID, accountID, err := store.GetOrCreateUser(r.Context(), ghUser.ID, ghUser.Login, ghUser.Email)
        if err != nil {
            log.Printf("Error creating user: %v", err)
            http.Error(w, "Failed to create user", http.StatusInternalServerError)
            return
        }

        // Create JWT token
        claims := VyalaClaims{
            UserID:    userID.String(),
            AccountID: accountID.String(),
            RegisteredClaims: jwt.RegisteredClaims{
                ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
            },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        tokenString, err := token.SignedString([]byte(sessionSecret))
        if err != nil {
            http.Error(w, "Failed to generate token", http.StatusInternalServerError)
            return
        }

        http.SetCookie(w, &http.Cookie{
            Name:     "vyala_token",
            Value:    tokenString,
            Path:     "/",
            HttpOnly: true,
            Secure:   isProduction,
            MaxAge:   7 * 24 * 60 * 60,
            SameSite: http.SameSiteLaxMode,
        })
        http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
    })

    // --- Auth Middleware ---
    authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("vyala_token")
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            claims := &VyalaClaims{}
            token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method")
                }
                return []byte(sessionSecret), nil
            })

            if err != nil || !token.Valid {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            accountID, err := uuid.Parse(claims.AccountID)
            if err != nil {
                http.Error(w, "Invalid token claims", http.StatusUnauthorized)
                return
            }

            // Inject account_id into context
            ctx := context.WithValue(r.Context(), "account_id", accountID)
            next.ServeHTTP(w, r.WithContext(ctx))
        }
    }

    // --- API Routes ---

    // POST /v1/scans - Ingest scan results (Authenticated via CLI API Key)
    mux.HandleFunc("POST /v1/scans", func(w http.ResponseWriter, r *http.Request) {
        // Limit payload to 10MB
        r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

        // Require API key
        if r.Header.Get("Authorization") != "Bearer "+cliAPIKey {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        var req backend.ScanRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        if req.RepoFullName == "" || req.CommitSHA == "" || req.GithubRepoID == 0 {
            http.Error(w, "Missing required fields", http.StatusBadRequest)
            return
        }

        // For MVP, ingests go to a system account or the authenticated user's account.
        // If CLI uses a user-specific key later, we extract accountID from key.
        accountID := backend.AccountIDPlaceholder 

        scanID, err := store.IngestScan(r.Context(), req, accountID)
        if err != nil {
            log.Printf("Error ingesting scan: %v", err)
            http.Error(w, "Failed to ingest scan", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID.String()})
    })

    // GET /v1/me - Get current logged in user
    mux.HandleFunc("GET /v1/me", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
        accountID := r.Context().Value("account_id").(uuid.UUID)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "account_id": accountID.String(),
        })
    }))

    // GET /v1/repos - List repositories for the dashboard
    mux.HandleFunc("GET /v1/repos", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
        accountID := r.Context().Value("account_id").(uuid.UUID)
        repos, err := store.GetRepos(r.Context(), accountID)
        if err != nil {
            http.Error(w, "Failed to fetch repos", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(repos)
    }))

    // GET /v1/repos/{repo_id}/scans
    mux.HandleFunc("GET /v1/repos/{repo_id}/scans", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
        accountID := r.Context().Value("account_id").(uuid.UUID)
        repoIDStr := r.PathValue("repo_id")
        repoID, err := uuid.Parse(repoIDStr)
        if err != nil {
            http.Error(w, "Invalid repo ID", http.StatusBadRequest)
            return
        }
        scans, err := store.GetScans(r.Context(), repoID, accountID, 10)
        if err != nil {
            http.Error(w, "Failed to fetch scans", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(scans)
    }))

    // GET /v1/repos/{repo_id}/findings
    mux.HandleFunc("GET /v1/repos/{repo_id}/findings", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
        accountID := r.Context().Value("account_id").(uuid.UUID)
        repoIDStr := r.PathValue("repo_id")
        repoID, err := uuid.Parse(repoIDStr)
        if err != nil {
            http.Error(w, "Invalid repo ID", http.StatusBadRequest)
            return
        }
        findings, err := store.GetOpenFindings(r.Context(), repoID, accountID)
        if err != nil {
            http.Error(w, "Failed to fetch findings", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(findings)
    }))

    // GET /v1/repos/{repo_id}/trends
    mux.HandleFunc("GET /v1/repos/{repo_id}/trends", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
        accountID := r.Context().Value("account_id").(uuid.UUID)
        repoIDStr := r.PathValue("repo_id")
        repoID, err := uuid.Parse(repoIDStr)
        if err != nil {
            http.Error(w, "Invalid repo ID", http.StatusBadRequest)
            return
        }
        trends, err := store.GetTrends(r.Context(), repoID, accountID, 90)
        if err != nil {
            http.Error(w, "Failed to fetch trends", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(trends)
    }))

    // CORS + Timeouts
    corsMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", frontendURL)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            next.ServeHTTP(w, r)
        })
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      corsMiddleware(mux),
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 60 * time.Second,
        IdleTimeout:  90 * time.Second,
    }

    // Graceful shutdown
    go func() {
        log.Printf("VYALA backend listening on port %s", port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown: ", err)
    }
}
package ghcomment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"vyala/internal/findings"
)

// fakeGitHub emulates just enough of the issues/comments API to exercise
// PostOrUpdate's find-or-create logic.
type fakeGitHub struct {
	mu       sync.Mutex
	comments map[int64]string
	nextID   int64
	creates  int
	updates  int
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{comments: map[int64]string{}, nextID: 1}
}

func (f *fakeGitHub) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet:
			var list []ghComment
			for id, body := range f.comments {
				list = append(list, ghComment{ID: id, Body: body})
			}
			json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost:
			var body ghComment
			json.NewDecoder(r.Body).Decode(&body)
			id := f.nextID
			f.nextID++
			f.comments[id] = body.Body
			f.creates++
			json.NewEncoder(w).Encode(ghComment{ID: id, Body: body.Body})
		case r.Method == http.MethodPatch:
			var body ghComment
			json.NewDecoder(r.Body).Decode(&body)
			// crude: id is last path segment
			idStr := r.URL.Path[len(r.URL.Path)-1:]
			id, _ := strconv.ParseInt(idStr, 10, 64)
			f.comments[id] = body.Body
			f.updates++
			json.NewEncoder(w).Encode(ghComment{ID: id, Body: body.Body})
		}
	}
}

func TestPostOrUpdate_CreatesOnceThenUpdates(t *testing.T) {
	fake := newFakeGitHub()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	cfg := Config{Token: "fake", APIBase: srv.URL, Repo: "acme/widgets", Client: srv.Client()}

	cbom := findings.CBOM{Version: "1.0", Findings: []findings.Finding{
		{File: "api/handlers.py", Line: 8, Algorithm: "RSA", Severity: "high",
			SuggestedReplacement: "Use ML-KEM hybrid"},
	}}
	body1 := RenderComment(cbom, cfg.Repo, "sha1", "medium")

	if err := PostOrUpdate(cfg, 42, body1); err != nil {
		t.Fatalf("first post: %v", err)
	}
	if fake.creates != 1 || fake.updates != 0 {
		t.Fatalf("expected 1 create, 0 updates after first run; got creates=%d updates=%d", fake.creates, fake.updates)
	}

	// Simulate a second Action run on a new push -- same PR, new SHA, same finding.
	body2 := RenderComment(cbom, cfg.Repo, "sha2", "medium")
	if err := PostOrUpdate(cfg, 42, body2); err != nil {
		t.Fatalf("second post: %v", err)
	}
	if fake.creates != 1 || fake.updates != 1 {
		t.Fatalf("expected still 1 create but 1 update after second run (dedup); got creates=%d updates=%d", fake.creates, fake.updates)
	}

	if len(fake.comments) != 1 {
		t.Fatalf("expected exactly 1 comment to exist on the PR, got %d", len(fake.comments))
	}
}

func TestRenderComment_NoFindings(t *testing.T) {
	cbom := findings.CBOM{Version: "1.0"}
	body := RenderComment(cbom, "acme/widgets", "sha1", "medium")
	if !containsMarker(body) {
		t.Fatal("comment must always contain the hidden dedup marker, even with zero findings")
	}
}

func containsMarker(s string) bool {
	return len(s) > 0 && (func() bool {
		return (len(s) >= len(Marker)) && (indexOf(s, Marker) >= 0)
	})()
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

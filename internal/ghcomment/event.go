package ghcomment

import (
	"encoding/json"
	"fmt"
	"os"
)

type PREvent struct {
	Number      int `json:"number"`
	PullRequest struct {
		Number int `json:"number"`
		Base   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func LoadEventFromEnv() (*PREvent, error) {
	path := os.Getenv("GITHUB_EVENT_PATH")
	if path == "" {
		return nil, fmt.Errorf("GITHUB_EVENT_PATH not set -- are we running inside a GitHub Action?")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading GITHUB_EVENT_PATH: %w", err)
	}
	var ev PREvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("parsing event payload: %w", err)
	}
	if ev.PullRequest.Number == 0 {
		return nil, fmt.Errorf("event payload has no pull_request block -- vyala's --post-pr-comment mode only supports pull_request / pull_request_target events")
	}
	return &ev, nil
}
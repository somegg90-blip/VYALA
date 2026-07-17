package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const SchemaVersion = "1.0"

var SeverityRank = map[string]int{"high": 0, "medium": 1, "low": 2}

func MeetsThreshold(sev, threshold string) bool {
	sr, ok1 := SeverityRank[sev]
	tr, ok2 := SeverityRank[threshold]
	if !ok1 || !ok2 {
		return false
	}
	return sr <= tr
}

type Finding struct {
	ID                    string `json:"finding_id"`
	File                  string `json:"file"`
	Line                  int    `json:"line"`
	Algorithm             string `json:"algorithm"`
	Severity              string `json:"severity"`
	Category              string `json:"category"`
	ExposureEstimate      string `json:"hnd_exposure_estimate"`
	SuggestedReplacement  string `json:"suggested_replacement"`
	RuleID                string `json:"rule_id"`
}

type CBOM struct {
	Version   string    `json:"version"`
	Generated time.Time `json:"generated_at"`
	Findings  []Finding `json:"findings"`
}

func GenerateFindingID(ruleID, file string, line int) string {
	h := sha256.New()
	h.Write([]byte(ruleID))
	h.Write([]byte{0})
	h.Write([]byte(file))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d", line)))
	sum := h.Sum(nil)
	return "vy-" + hex.EncodeToString(sum)[:16]
}
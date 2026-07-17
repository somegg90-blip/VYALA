package severity

import "strings"

var highRiskIndicators = []string{
	"/api/", "/public/", "/handlers/", "/handler/",
	"auth", "payment", "pii", "/routes/", "/controllers/",
}

func Classify(file string) string {
	lower := "/" + strings.ToLower(strings.TrimPrefix(file, "/"))
	for _, ind := range highRiskIndicators {
		if strings.Contains(lower, ind) {
			return "high"
		}
	}
	return "medium"
}

func ExposureEstimate(file, sev string) string {
	if sev == "high" {
		return "External-facing or sensitive-data path -- treat as higher HNDL exposure"
	}
	return "Internal / not matched to an external-facing path convention"
}
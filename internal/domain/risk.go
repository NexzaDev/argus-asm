package domain

import "fmt"

const (
	maxRiskScore       = 100
	exposureStep       = 2
	exposureServiceCap = 5
	agingThresholdDays = 30
	agingBonus         = 5
)

type ExposureContext struct {
	OpenServices int
	DaysOpen     int
}

type RiskAssessment struct {
	Score   int      `json:"score"`
	Factors []string `json:"factors"`
}

func AssessRisk(severity Severity, status FindingStatus, reason string, exposure ExposureContext) RiskAssessment {
	score := severity.Weight()
	factors := []string{fmt.Sprintf("severity %s contributes %d", severity, score)}

	if exposure.OpenServices > 0 {
		counted := exposure.OpenServices
		if counted > exposureServiceCap {
			counted = exposureServiceCap
		}
		bonus := counted * exposureStep
		score += bonus
		factors = append(factors, fmt.Sprintf("asset has %d open service(s) contributes %d", exposure.OpenServices, bonus))
	}
	if exposure.DaysOpen >= agingThresholdDays {
		score += agingBonus
		factors = append(factors, fmt.Sprintf("open for %d days (threshold %d) contributes %d", exposure.DaysOpen, agingThresholdDays, agingBonus))
	}
	if score > maxRiskScore {
		score = maxRiskScore
		factors = append(factors, fmt.Sprintf("score capped at %d", maxRiskScore))
	}

	switch status {
	case FindingAccepted:
		score = 0
		factors = append(factors, fmt.Sprintf("risk accepted, score set to 0: %s", reason))
	case FindingResolved:
		score = 0
		factors = append(factors, "resolved, score set to 0")
	}
	return RiskAssessment{Score: score, Factors: factors}
}

package domain

import "time"

type RuleHit struct {
	Rule        string `json:"rule"`
	Matched     bool   `json:"matched"`
	Explanation string `json:"explanation"`
	Score       int    `json:"score"`
}

type RiskResult struct {
	ObservationID         string    `json:"observation_id"`
	BladeZoneID           string    `json:"blade_zone_id"`
	Level                 string    `json:"level"`
	Score                 int       `json:"score"`
	Blocked               bool      `json:"blocked"`
	Reasons               []string  `json:"reasons"`
	SuggestedRetestPoints []string  `json:"suggested_retest_points"`
	RuleHits              []RuleHit `json:"rule_hits"`
}

type AssessmentSnapshot struct {
	TaskID        string            `json:"task_id"`
	RuleVersion   string            `json:"rule_version"`
	Results       []RiskResult      `json:"results"`
	HighestLevel  string            `json:"highest_level"`
	CreatedAt     time.Time         `json:"created_at"`
	ZoneSummaries []RiskZoneSummary `json:"zone_summaries,omitempty"`
}

type RiskZoneSummary struct {
	BladeZoneID         string `json:"blade_zone_id"`
	Low                 int    `json:"low"`
	Medium              int    `json:"medium"`
	High                int    `json:"high"`
	Critical            int    `json:"critical"`
	TotalScore          int    `json:"total_score"`
	BlockedObservations int    `json:"blocked_observations"`
}

type ReleaseCredential struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"task_id"`
	RiskSummary      string    `json:"risk_summary"`
	EvidenceDigest   string    `json:"evidence_digest"`
	Reviewer         string    `json:"reviewer"`
	Decision         string    `json:"decision"`
	IssuedAt         time.Time `json:"issued_at"`
	CredentialDigest string    `json:"credential_digest"`
}

func (c ReleaseCredential) Validate() error {
	if err := Required("reviewer", c.Reviewer); err != nil {
		return err
	}
	if c.Decision != "approved" {
		return ValidationError{"decision", "仅能签发通过决定"}
	}
	if len(c.EvidenceDigest) < 16 || len(c.CredentialDigest) < 16 {
		return ValidationError{"digest", "凭证摘要无效"}
	}
	return nil
}

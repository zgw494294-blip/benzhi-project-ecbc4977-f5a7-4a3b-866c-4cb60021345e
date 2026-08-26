package application

import (
	"bladeready/internal/assessment"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

func digest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) Release(ctx context.Context, id string, r ReleaseRequest) (store.TaskBundle, error) {
	unlock, err := s.lockTask(ctx, id)
	if err != nil {
		return store.TaskBundle{}, err
	}
	defer unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusReviewing {
		return b, domain.ErrInvalidTransition
	}
	boundaryDigest := b.BoundaryDigest
	if boundaryDigest == "" {
		boundaryDigest = b.BoundarySummary
	}
	if err = domain.ValidateBoundaryIntegrity(b.Task.BladeCount, b.Zones, boundaryDigest, b.ZoneCoverage); err != nil {
		return b, err
	}
	if b.Assessment == nil || b.Assessment.RuleVersion != assessment.RuleVersion {
		return b, domain.ValidationError{Field: "assessment", Message: "风险规则版本与冻结快照不一致"}
	}
	if !r.ConfirmBoundary || !r.ConfirmRetests || !r.ConfirmAudit {
		return b, domain.ValidationError{Field: "confirmations", Message: "必须确认边界、复测与审计证据"}
	}
	if err = domain.Required("reviewer", r.Reviewer); err != nil {
		return b, err
	}
	if err = domain.ValidateReviewEvidence(b.Retests, b.Deviations); err != nil {
		return b, err
	}
	evidence := digest(map[string]any{"zones": b.Zones, "observations": b.Observations, "assessment": b.Assessment, "repairs": b.RepairPlan, "retests": b.Retests, "deviations": b.Deviations})
	risk := "无风险快照"
	if b.Assessment != nil {
		risk = assessment.Explain(*b.Assessment).Text() + "；规则=" + b.Assessment.RuleVersion
	}
	c := domain.ReleaseCredential{ID: newID("credential"), TaskID: id, RiskSummary: risk, EvidenceDigest: evidence, Reviewer: r.Reviewer, Decision: "approved", IssuedAt: time.Now().UTC()}
	c.CredentialDigest = digest(map[string]any{"task": id, "risk": risk, "evidence": evidence, "reviewer": r.Reviewer, "issued_at": c.IssuedAt})
	if err = c.Validate(); err != nil {
		return b, err
	}
	b.Credential = &c
	if err = b.Task.Move(r.ExpectedVersion, domain.StatusReleased); err != nil {
		return b, err
	}
	e, _ := event(id, "release.issued", r.Reviewer, b.Task.Version, c)
	return save(ctx, s.repo, b, e, r.IdempotencyKey)
}

func RiskSummary(b store.TaskBundle) string {
	if b.Assessment == nil {
		return "尚未评估"
	}
	levels := map[string]int{}
	for _, r := range b.Assessment.Results {
		levels[r.Level]++
	}
	parts := []string{"最高风险 " + b.Assessment.HighestLevel}
	for _, l := range []string{"critical", "high", "medium", "low"} {
		if levels[l] > 0 {
			parts = append(parts, l+"="+fmtInt(levels[l]))
		}
	}
	return strings.Join(parts, "，")
}
func fmtInt(v int) string {
	const digits = "0123456789"
	if v < 10 {
		return string(digits[v])
	}
	return "多项"
}

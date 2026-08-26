package assessment

import (
	"bladeready/internal/domain"
	"fmt"
	"strings"
)

const RuleVersion = "blade-risk-2026.1"

type Context struct {
	Zone        domain.BladeZone
	Observation domain.DroneObservation
	PlannedWind float64
}

type Rule interface{ Evaluate(Context) domain.RuleHit }

type sizeRule struct{}

func (sizeRule) Evaluate(c Context) domain.RuleHit {
	limit := c.Zone.MaxCrackMM
	label := "裂纹长度限值"
	if c.Observation.DefectType == domain.DefectDelamination {
		limit, label = c.Zone.MaxDelaminationMM, "分层长度限值"
	}
	ratio := c.Observation.LengthMM / limit
	hit := domain.RuleHit{Rule: "size_limit", Explanation: fmt.Sprintf("%s %.1fmm，观测 %.1fmm", label, limit, c.Observation.LengthMM)}
	switch {
	case ratio > 1:
		hit.Matched, hit.Score = true, 55
	case ratio >= .75:
		hit.Matched, hit.Score = true, 32
	case ratio >= .5:
		hit.Matched, hit.Score = true, 16
	}
	return hit
}

type defectRule struct{}

func (defectRule) Evaluate(c Context) domain.RuleHit {
	h := domain.RuleHit{Rule: "defect_type", Explanation: "缺陷类型基础风险"}
	switch c.Observation.DefectType {
	case domain.DefectLightning:
		h.Matched, h.Score, h.Explanation = true, 45, "雷击损伤需要结构复核"
	case domain.DefectCrack:
		h.Matched, h.Score, h.Explanation = true, 25, "裂纹可能继续扩展"
	case domain.DefectDelamination:
		h.Matched, h.Score, h.Explanation = true, 30, "分层影响复合材料承载"
	case domain.DefectErosion:
		h.Matched, h.Score, h.Explanation = true, 10, "前缘侵蚀影响气动性能"
	}
	return h
}

type positionRule struct{}

func (positionRule) Evaluate(c Context) domain.RuleHit {
	h := domain.RuleHit{Rule: "critical_position", Explanation: "缺陷不在根部关键区域"}
	code := strings.ToLower(c.Zone.ZoneCode)
	if strings.Contains(code, "root") || strings.Contains(c.Zone.ZoneCode, "根") || c.Observation.PositionMM < 3000 {
		h.Matched, h.Score, h.Explanation = true, 25, "缺陷靠近叶根高载荷区域"
	}
	return h
}

type windRule struct{}

func (windRule) Evaluate(c Context) domain.RuleHit {
	h := domain.RuleHit{Rule: "wind_boundary", Explanation: fmt.Sprintf("计划风速 %.1fm/s，边界 %.1fm/s", c.PlannedWind, c.Zone.OperatingWindLimit)}
	if c.PlannedWind > c.Zone.OperatingWindLimit {
		h.Matched, h.Score, h.Explanation = true, 40, h.Explanation+"，超出运行边界"
	}
	return h
}

func DefaultRules() []Rule { return []Rule{sizeRule{}, defectRule{}, positionRule{}, windRule{}} }

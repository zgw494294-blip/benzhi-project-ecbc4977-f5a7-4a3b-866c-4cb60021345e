package assessment

import (
	"bladeready/internal/domain"
	"fmt"
	"sort"
	"strings"
)

type Summary struct {
	HighestLevel    string         `json:"highest_level"`
	BlockedCount    int            `json:"blocked_count"`
	LevelCounts     map[string]int `json:"level_counts"`
	BlockingReasons []string       `json:"blocking_reasons"`
	RetestPoints    []string       `json:"retest_points"`
}

// Explain folds individual rule results into a deterministic safety summary.
// Duplicate reasons and retest points are removed without losing useful text.
func Explain(snapshot domain.AssessmentSnapshot) Summary {
	summary := Summary{HighestLevel: snapshot.HighestLevel, LevelCounts: map[string]int{}}
	reasons, points := map[string]bool{}, map[string]bool{}
	for _, result := range snapshot.Results {
		summary.LevelCounts[result.Level]++
		if result.Blocked {
			summary.BlockedCount++
		}
		for _, reason := range result.Reasons {
			if strings.TrimSpace(reason) != "" {
				reasons[reason] = true
			}
		}
		for _, point := range result.SuggestedRetestPoints {
			if strings.TrimSpace(point) != "" {
				points[point] = true
			}
		}
	}
	for reason := range reasons {
		summary.BlockingReasons = append(summary.BlockingReasons, reason)
	}
	for point := range points {
		summary.RetestPoints = append(summary.RetestPoints, point)
	}
	sort.Strings(summary.BlockingReasons)
	sort.Strings(summary.RetestPoints)
	return summary
}

func (s Summary) Text() string {
	if len(s.LevelCounts) == 0 {
		return "没有风险评估结果"
	}
	base := fmt.Sprintf("最高风险 %s，阻断观测 %d 项", s.HighestLevel, s.BlockedCount)
	if len(s.BlockingReasons) == 0 {
		return base + "，未命中阻断原因"
	}
	return base + "；" + strings.Join(s.BlockingReasons, "；")
}

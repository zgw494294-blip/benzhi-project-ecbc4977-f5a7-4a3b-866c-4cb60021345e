package assessment

import (
	"bladeready/internal/domain"
	"fmt"
	"sort"
	"time"
)

type Engine struct{ rules []Rule }

func NewEngine() *Engine { return &Engine{rules: DefaultRules()} }

func (e *Engine) Evaluate(zone domain.BladeZone, observation domain.DroneObservation, plannedWind float64) domain.RiskResult {
	ctx := Context{Zone: zone, Observation: observation, PlannedWind: plannedWind}
	result := domain.RiskResult{ObservationID: observation.ID, BladeZoneID: zone.ID}
	for _, rule := range e.rules {
		hit := rule.Evaluate(ctx)
		result.RuleHits = append(result.RuleHits, hit)
		result.Score += hit.Score
		if hit.Matched && hit.Score >= 30 {
			result.Reasons = append(result.Reasons, hit.Explanation)
		}
	}
	switch {
	case result.Score >= 80:
		result.Level, result.Blocked = "critical", true
	case result.Score >= 50:
		result.Level, result.Blocked = "high", true
	case result.Score >= 25:
		result.Level = "medium"
	default:
		result.Level = "low"
	}
	result.SuggestedRetestPoints = []string{
		fmt.Sprintf("%s@%.0fmm 主缺陷尺寸", zone.ZoneCode, observation.PositionMM),
		fmt.Sprintf("%s@%.0fmm 周边层合完整性", zone.ZoneCode, observation.PositionMM+100),
	}
	return result
}

func (e *Engine) Snapshot(taskID string, zones []domain.BladeZone, observations []domain.DroneObservation, wind float64) domain.AssessmentSnapshot {
	s, _ := e.SnapshotChecked(taskID, zones, observations, wind)
	return s
}

func (e *Engine) SnapshotChecked(taskID string, zones []domain.BladeZone, observations []domain.DroneObservation, wind float64) (domain.AssessmentSnapshot, error) {
	byID := make(map[string]domain.BladeZone)
	for _, z := range zones {
		byID[z.ID] = z
	}
	s := domain.AssessmentSnapshot{TaskID: taskID, RuleVersion: RuleVersion, HighestLevel: "low", CreatedAt: time.Now().UTC()}
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	stats := make(map[string]*domain.RiskZoneSummary)
	for _, o := range observations {
		z, ok := byID[o.BladeZoneID]
		if !ok {
			return domain.AssessmentSnapshot{}, fmt.Errorf("观测 %s 引用了不存在的分区", o.ID)
		}
		r := e.Evaluate(z, o, wind)
		s.Results = append(s.Results, r)
		stat := stats[z.ID]
		if stat == nil {
			stat = &domain.RiskZoneSummary{BladeZoneID: z.ID}
			stats[z.ID] = stat
		}
		stat.TotalScore += r.Score
		if r.Blocked {
			stat.BlockedObservations++
		}
		switch r.Level {
		case "low":
			stat.Low++
		case "medium":
			stat.Medium++
		case "high":
			stat.High++
		case "critical":
			stat.Critical++
		}
		if rank[r.Level] > rank[s.HighestLevel] {
			s.HighestLevel = r.Level
		}
	}
	for _, stat := range stats {
		s.ZoneSummaries = append(s.ZoneSummaries, *stat)
	}
	sort.Slice(s.ZoneSummaries, func(i, j int) bool { return s.ZoneSummaries[i].BladeZoneID < s.ZoneSummaries[j].BladeZoneID })
	return s, nil
}

package application

import (
	"bladeready/internal/assessment"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"time"
)

func (s *Service) Assess(ctx context.Context, id string, r AssessRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusObserving {
		return b, domain.ErrInvalidTransition
	}
	digest := b.BoundaryDigest
	if digest == "" {
		digest = b.BoundarySummary
	}
	if err = domain.ValidateBoundaryIntegrity(b.Task.BladeCount, b.Zones, digest, b.ZoneCoverage); err != nil {
		return b, err
	}
	if len(b.Observations) == 0 {
		return b, domain.ValidationError{Field: "observations", Message: "没有可评估观测"}
	}
	if b.Assessment != nil && b.Assessment.RuleVersion != assessment.RuleVersion {
		return b, domain.ValidationError{Field: "rule_version", Message: "规则版本与当前评估引擎不一致"}
	}
	snapshot, err := s.engine.SnapshotChecked(id, b.Zones, b.Observations, r.PlannedWind)
	if err != nil {
		return b, err
	}
	b.Assessment = &snapshot
	if err = b.Task.Move(r.ExpectedVersion, domain.StatusAssessed); err != nil {
		return b, err
	}
	e, _ := event(id, "risk.assessed", r.Actor, b.Task.Version, snapshot)
	return s.save(ctx, b, e, r.IdempotencyKey)
}

func (s *Service) FreezeRepairPlan(ctx context.Context, id string, r FreezeRepairPlanRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusAssessed {
		return b, domain.ErrInvalidTransition
	}
	if b.Assessment == nil {
		return b, domain.ValidationError{Field: "assessment", Message: "缺少风险快照"}
	}
	if b.Assessment.RuleVersion != assessment.RuleVersion {
		return b, domain.ValidationError{Field: "rule_version", Message: "风险规则版本与当前评估引擎不一致"}
	}
	for i := range r.Actions {
		a := &r.Actions[i]
		if a.ID == "" {
			a.ID = newID("repair")
		}
		a.TaskID = id
		if err = a.Validate(); err != nil {
			return b, err
		}
	}
	if len(r.Actions) == 0 {
		return b, domain.ValidationError{Field: "actions", Message: "维修复核计划不能为空"}
	}
	if err = domain.ValidateRepairCoverage(b.Assessment.Results, r.Actions); err != nil {
		return b, err
	}
	b.RepairPlan = r.Actions
	if err = b.Task.Move(r.ExpectedVersion, domain.StatusRepairing); err != nil {
		return b, err
	}
	e, _ := event(id, "repair_plan.frozen", r.Actor, b.Task.Version, r.Actions)
	return s.save(ctx, b, e, r.IdempotencyKey)
}

func (s *Service) AddRetests(ctx context.Context, id string, r AddRetestsRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusRepairing && b.Task.Status != domain.StatusRetesting {
		return b, domain.ErrInvalidTransition
	}
	if len(r.Readings) == 0 {
		return b, domain.ValidationError{Field: "readings", Message: "复测读数不能为空"}
	}
	seenBatch := make(map[string]bool)
	for i := range r.Readings {
		x := &r.Readings[i]
		if seenBatch[x.ObservationID] {
			return b, domain.ValidationError{Field: "readings", Message: "同一复测批次不能重复观测"}
		}
		seenBatch[x.ObservationID] = true
		x.ID = newID("retest")
		x.TaskID = id
		if err = x.Validate(); err != nil {
			return b, err
		}
		valid := false
		for _, o := range b.Observations {
			if o.ID == x.ObservationID && o.BladeZoneID == x.BladeZoneID {
				valid = true
			}
		}
		if !valid {
			return b, domain.ValidationError{Field: "observation_id", Message: "复测只能针对对应原观测分区"}
		}
		if latest, ok := latestRetest(b.Retests, x.ObservationID); ok && !x.MeasuredAt.After(latest.MeasuredAt) {
			return b, domain.ValidationError{Field: "measured_at", Message: "复测时间必须晚于已有最新读数"}
		}
	}
	b.Retests = append(b.Retests, r.Readings...)
	measured := make(map[string]time.Time)
	for _, reading := range r.Readings {
		measured[reading.ObservationID] = reading.MeasuredAt
	}
	for i := range b.RepairPlan {
		if completedAt, ok := measured[b.RepairPlan[i].ObservationID]; ok && b.RepairPlan[i].CompletedAt == nil {
			b.RepairPlan[i].CompletedAt = &completedAt
		}
	}
	detected := s.comparator.Compare(id, b.Observations, b.Retests)
	b.Deviations = mergeDeviations(b.Deviations, detected)
	if b.Task.Status == domain.StatusRepairing {
		if err = b.Task.Move(r.ExpectedVersion, domain.StatusRetesting); err != nil {
			return b, err
		}
	} else {
		b.Task.Version++
	}
	e, _ := event(id, "retests.recorded", r.Actor, b.Task.Version, map[string]any{"readings": r.Readings, "deviations": b.Deviations})
	return s.save(ctx, b, e, r.IdempotencyKey)
}

func (s *Service) CloseDeviation(ctx context.Context, id, deviationID string, r CloseDeviationRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusRetesting {
		return b, domain.ErrInvalidTransition
	}
	found := false
	for i := range b.Deviations {
		if b.Deviations[i].ID == deviationID {
			found = true
			original, ok := findObservation(b.Observations, b.Deviations[i].ObservationID)
			if !ok {
				return b, domain.ValidationError{Field: "deviation", Message: "找不到偏差对应的原观测"}
			}
			latest, ok := latestRetest(b.Retests, original.ID)
			if !ok || latest.BladeZoneID != b.Deviations[i].BladeZoneID {
				return b, domain.ValidationError{Field: "retest", Message: "必须先在偏差对应分区完成定向复测"}
			}
			if remaining := s.comparator.Compare(id, []domain.DroneObservation{original}, []domain.RetestReading{latest}); len(remaining) > 0 {
				return b, domain.ValidationError{Field: "retest", Message: "最新定向复测仍不满足关闭条件: " + remaining[0].Reason}
			}
			if err = b.Deviations[i].Close(r.CorrectiveAction, r.Actor, time.Now().UTC()); err != nil {
				return b, err
			}
		}
	}
	if !found {
		return b, domain.ErrNotFound
	}
	b.Task.Version++
	e, _ := event(id, "deviation.closed", r.Actor, b.Task.Version, map[string]string{"deviation_id": deviationID, "action": r.CorrectiveAction})
	return s.save(ctx, b, e, r.IdempotencyKey)
}

func mergeDeviations(existing, detected []domain.Deviation) []domain.Deviation {
	result := make([]domain.Deviation, 0, len(existing)+len(detected))
	known := make(map[string]bool)
	for _, deviation := range existing {
		key := deviation.ObservationID + "/" + deviation.Kind
		known[key] = true
		result = append(result, deviation)
	}
	for _, deviation := range detected {
		key := deviation.ObservationID + "/" + deviation.Kind
		if known[key] {
			continue
		}
		deviation.ID = newID("dev")
		result = append(result, deviation)
	}
	return result
}

func findObservation(items []domain.DroneObservation, id string) (domain.DroneObservation, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return domain.DroneObservation{}, false
}

func latestRetest(items []domain.RetestReading, observationID string) (domain.RetestReading, bool) {
	var latest domain.RetestReading
	found := false
	for _, item := range items {
		if item.ObservationID == observationID && (!found || item.MeasuredAt.After(latest.MeasuredAt)) {
			latest, found = item, true
		}
	}
	return latest, found
}

func (s *Service) PrepareReview(ctx context.Context, id string, r PrepareReviewRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusRetesting {
		return b, domain.ErrInvalidTransition
	}
	if err = domain.ValidateReviewEvidence(b.Retests, b.Deviations); err != nil {
		return b, err
	}
	for _, action := range b.RepairPlan {
		original, ok := findObservation(b.Observations, action.ObservationID)
		if !ok {
			return b, domain.ValidationError{Field: "repair_plan", Message: "维修动作缺少对应观测"}
		}
		latest, ok := latestRetest(b.Retests, original.ID)
		if !ok || len(s.comparator.Compare(id, []domain.DroneObservation{original}, []domain.RetestReading{latest})) > 0 {
			return b, domain.ValidationError{Field: "retests", Message: "每个维修动作都必须有通过的对应复测"}
		}
	}
	if err = b.Task.Move(r.ExpectedVersion, domain.StatusReviewing); err != nil {
		return b, err
	}
	e, _ := event(id, "safety_review.prepared", r.Actor, b.Task.Version, map[string]int{"retests": len(b.Retests)})
	return s.save(ctx, b, e, r.IdempotencyKey)
}

package application

import (
	"bladeready/internal/assessment"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

type Service struct {
	repo       store.Repository
	engine     *assessment.Engine
	comparator *assessment.Comparator
	mu         sync.Mutex
	cacheMu    sync.RWMutex
	released   map[string]store.TaskBundle
}

func New(repo store.Repository) *Service {
	return &Service{
		repo:       repo,
		engine:     assessment.NewEngine(),
		comparator: assessment.NewComparator(),
		released:   make(map[string]store.TaskBundle),
	}
}
func (s *Service) Repository() store.Repository { return s.repo }
func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}
func response(v any) []byte { b, _ := json.Marshal(v); return b }
func event(taskID, typ, actor string, version int64, payload any) (domain.Event, error) {
	return domain.NewEvent(newID("evt"), taskID, typ, actor, version, payload)
}

func (s *Service) List(ctx context.Context, filters ...store.TaskListFilter) ([]domain.InspectionTask, error) {
	if len(filters) > 0 {
		return s.ListFiltered(ctx, filters[0])
	}
	return s.repo.ListTasks(ctx)
}
func (s *Service) ListFiltered(ctx context.Context, filter store.TaskListFilter) ([]domain.InspectionTask, error) {
	if filtered, ok := s.repo.(interface {
		ListTasksFiltered(context.Context, store.TaskListFilter) ([]domain.InspectionTask, error)
	}); ok {
		return filtered.ListTasksFiltered(ctx, filter)
	}
	return s.repo.ListTasks(ctx)
}
func (s *Service) Get(ctx context.Context, id string) (store.TaskBundle, error) {
	s.cacheMu.RLock()
	b, ok := s.released[id]
	s.cacheMu.RUnlock()
	if ok {
		return b, nil
	}
	b, err := s.repo.LoadTask(ctx, id)
	if err != nil {
		return store.TaskBundle{}, err
	}
	if b.Task.Status == domain.StatusReleased {
		s.cacheMu.Lock()
		s.released[id] = b
		s.cacheMu.Unlock()
	}
	return b, nil
}
func (s *Service) Audit(ctx context.Context, id string) ([]domain.Event, error) {
	return s.repo.Events(ctx, id)
}

func (s *Service) CreateTask(ctx context.Context, r CreateTaskRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok, err := s.repo.Idempotent(ctx, r.IdempotencyKey); err != nil {
		return store.TaskBundle{}, err
	} else if ok {
		var b store.TaskBundle
		err = json.Unmarshal(cached, &b)
		return b, err
	}
	t, err := domain.NewTask(newID("task"), r.WindFarm, r.TurbineCode, r.BladeCount, r.InspectionWindowStart, r.InspectionWindowEnd, r.CreatedBy)
	if err != nil {
		return store.TaskBundle{}, err
	}
	existing, err := s.repo.ListTasks(ctx)
	if err != nil {
		return store.TaskBundle{}, err
	}
	for _, other := range existing {
		if other.TurbineCode == t.TurbineCode && t.InspectionWindowStart.Before(other.InspectionWindowEnd) && other.InspectionWindowStart.Before(t.InspectionWindowEnd) {
			return store.TaskBundle{}, domain.ValidationError{Field: "inspection_window", Message: "同一风机存在重叠巡检窗口"}
		}
	}
	b := store.TaskBundle{Task: *t, Zones: []domain.BladeZone{}, Observations: []domain.DroneObservation{}, RepairPlan: []domain.RepairAction{}, Retests: []domain.RetestReading{}, Deviations: []domain.Deviation{}}
	e, err := event(t.ID, "task.created", r.CreatedBy, t.Version, t)
	if err != nil {
		return b, err
	}
	_, err = s.repo.CreateTask(ctx, *t, e, r.IdempotencyKey, response(b))
	return b, err
}

func loadForWrite(ctx context.Context, repo store.Repository, id string, expected int64) (store.TaskBundle, error) {
	b, err := repo.LoadTask(ctx, id)
	if err != nil {
		return b, err
	}
	if err = b.Task.CheckVersion(expected); err != nil {
		return b, err
	}
	return b, nil
}
func save(ctx context.Context, repo store.Repository, b store.TaskBundle, e domain.Event, key string) (store.TaskBundle, error) {
	raw := response(b)
	cached, err := repo.SaveBundle(ctx, b, e, key, raw)
	if err != nil {
		return store.TaskBundle{}, err
	}
	var result store.TaskBundle
	if err = json.Unmarshal(cached, &result); err != nil {
		return store.TaskBundle{}, err
	}
	return result, nil
}

func (s *Service) SetZones(ctx context.Context, id string, r SetZonesRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusDraft {
		return b, fmt.Errorf("%w: 仅草稿任务可登记边界", domain.ErrInvalidTransition)
	}
	if len(r.Zones) == 0 {
		return b, domain.ValidationError{Field: "zones", Message: "至少登记一个分区"}
	}
	now := b.Task.CreatedAt
	for i := range r.Zones {
		z := &r.Zones[i]
		z.TaskID = id
		if z.ID == "" {
			z.ID = newID("zone")
		}
		if err = z.Validate(b.Task.BladeCount); err != nil {
			return b, err
		}
		now = now.Add(1)
		if err = z.Freeze(now); err != nil {
			return b, err
		}
	}
	if err = domain.ValidateBoundarySet(b.Task.BladeCount, r.Zones); err != nil {
		return b, err
	}
	b.Zones = r.Zones
	b.BoundaryDigest = domain.BoundaryDigest(b.Zones)
	b.BoundarySummary = b.BoundaryDigest
	b.BoundaryFrozenAt = &now
	b.ZoneCoverage = domain.BoundaryCoverage(b.Task.BladeCount, b.Zones)
	if err = b.Task.Move(r.ExpectedVersion, domain.StatusObserving); err != nil {
		return b, err
	}
	e, _ := event(id, "zones.frozen", r.Actor, b.Task.Version, map[string]any{"zones": r.Zones, "boundary_digest": b.BoundaryDigest, "coverage": b.ZoneCoverage, "frozen_at": b.BoundaryFrozenAt})
	return save(ctx, s.repo, b, e, r.IdempotencyKey)
}

func (s *Service) AddObservation(ctx context.Context, id string, r AddObservationRequest) (store.TaskBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadForWrite(ctx, s.repo, id, r.ExpectedVersion)
	if err != nil {
		return b, err
	}
	if b.Task.Status != domain.StatusObserving {
		return b, domain.ErrInvalidTransition
	}
	o := r.Observation
	o.ID = newID("obs")
	o.TaskID = id
	if err = o.Validate(); err != nil {
		return b, err
	}
	found := false
	for _, z := range b.Zones {
		if z.ID == o.BladeZoneID {
			found = true
		}
	}
	if !found {
		return b, domain.ValidationError{Field: "blade_zone_id", Message: "分区不属于该任务"}
	}
	for _, x := range b.Observations {
		if x.Sequence == o.Sequence {
			return b, domain.ValidationError{Field: "sequence", Message: "观测序号重复"}
		}
	}
	b.Observations = append(b.Observations, o)
	if err = domain.ValidateObservationTimeline(b.Observations); err != nil {
		return b, err
	}
	b.Task.Version++
	e, _ := event(id, "observation.added", r.Actor, b.Task.Version, o)
	return save(ctx, s.repo, b, e, r.IdempotencyKey)
}

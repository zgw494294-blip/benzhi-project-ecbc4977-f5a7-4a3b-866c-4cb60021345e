package taskgate_test

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type blockingRepository struct {
	bundle        store.TaskBundle
	loads         atomic.Int32
	canceledLoads atomic.Int32
	firstEntered  chan struct{}
	releaseFirst  chan struct{}
}

type errObservedContext struct {
	context.Context
	once       sync.Once
	errChecked chan struct{}
}

func (c *errObservedContext) Err() error {
	c.once.Do(func() { close(c.errChecked) })
	return c.Context.Err()
}

func newBlockingRepository(t *testing.T) *blockingRepository {
	t.Helper()
	now := time.Date(2026, time.August, 26, 8, 0, 0, 0, time.UTC)
	task, err := domain.NewTask("task-gate", "北场", "WT-17", 1, now, now.Add(2*time.Hour), "巡检员")
	if err != nil {
		t.Fatal(err)
	}
	return &blockingRepository{
		bundle: store.TaskBundle{
			Task:         *task,
			Zones:        []domain.BladeZone{},
			Observations: []domain.DroneObservation{},
			RepairPlan:   []domain.RepairAction{},
			Retests:      []domain.RetestReading{},
			Deviations:   []domain.Deviation{},
		},
		firstEntered: make(chan struct{}),
		releaseFirst: make(chan struct{}),
	}
}

func (r *blockingRepository) LoadTask(ctx context.Context, _ string) (store.TaskBundle, error) {
	if r.loads.Add(1) == 1 {
		close(r.firstEntered)
		<-r.releaseFirst
	}
	if err := ctx.Err(); err != nil {
		r.canceledLoads.Add(1)
		return store.TaskBundle{}, err
	}
	return r.bundle, nil
}

func (r *blockingRepository) SaveBundle(_ context.Context, _ store.TaskBundle, _ domain.Event, _ string, response []byte) ([]byte, error) {
	var saved store.TaskBundle
	if err := json.Unmarshal(response, &saved); err != nil {
		return nil, err
	}
	r.bundle = saved
	return response, nil
}

func (r *blockingRepository) CreateTask(context.Context, domain.InspectionTask, domain.Event, string, []byte) ([]byte, error) {
	return nil, errors.New("本测试不创建任务")
}

func (r *blockingRepository) ListTasks(context.Context) ([]domain.InspectionTask, error) {
	return []domain.InspectionTask{r.bundle.Task}, nil
}

func (r *blockingRepository) Events(context.Context, string) ([]domain.Event, error) {
	return nil, nil
}

func (r *blockingRepository) Idempotent(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (r *blockingRepository) Check(context.Context) error { return nil }
func (r *blockingRepository) Close() error                { return nil }

func TestCanceledTaskGateDoesNotEnterRepository(t *testing.T) {
	repo := newBlockingRepository(t)
	service := application.New(repo)
	taskID := repo.bundle.Task.ID
	request := application.SetZonesRequest{
		ExpectedVersion: 1,
		Actor:           "边界复核员",
		IdempotencyKey:  "gate-first",
		Zones: []domain.BladeZone{{
			BladeIndex:         1,
			ZoneCode:           "ROOT",
			MaterialType:       "玻纤",
			MaxCrackMM:         10,
			MaxDelaminationMM:  20,
			OperatingWindLimit: 14,
		}},
	}

	firstDone := make(chan error, 1)
	go func() {
		_, err := service.SetZones(context.Background(), taskID, request)
		firstDone <- err
	}()
	<-repo.firstEntered

	cancelBase, cancel := context.WithCancel(context.Background())
	canceled := &errObservedContext{Context: cancelBase, errChecked: make(chan struct{})}
	secondDone := make(chan error, 1)
	go func() {
		second := request
		second.IdempotencyKey = "gate-canceled"
		_, err := service.SetZones(canceled, taskID, second)
		secondDone <- err
	}()
	<-canceled.errChecked
	cancel()
	close(repo.releaseFirst)

	if err := <-firstDone; err != nil {
		t.Fatalf("占用任务门的首个命令失败: %v", err)
	}
	if err := <-secondDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("已取消命令应返回 context.Canceled，实际为 %v", err)
	}
	if got := repo.canceledLoads.Load(); got != 0 {
		t.Fatalf("已取消命令在任务门释放后仍进入 Repository.LoadTask，调用次数=%d", got)
	}
}

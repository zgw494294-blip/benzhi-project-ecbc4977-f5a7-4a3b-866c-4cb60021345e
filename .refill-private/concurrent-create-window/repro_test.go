package concurrentcreatewindow

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"sync"
	"testing"
	"time"
)

type repository struct {
	mu           sync.Mutex
	tasks        []domain.InspectionTask
	listCalls    int
	firstListed  chan struct{}
	secondListed chan struct{}
	allowFirst   chan struct{}
}

func newRepository() *repository {
	return &repository{
		firstListed:  make(chan struct{}),
		secondListed: make(chan struct{}),
		allowFirst:   make(chan struct{}),
	}
}

func (r *repository) CreateTask(_ context.Context, task domain.InspectionTask, _ domain.Event, _ string, response []byte) ([]byte, error) {
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
	return response, nil
}

func (r *repository) LoadTask(context.Context, string) (store.TaskBundle, error) {
	return store.TaskBundle{}, domain.ErrNotFound
}

func (r *repository) ListTasks(context.Context) ([]domain.InspectionTask, error) {
	r.mu.Lock()
	r.listCalls++
	call := r.listCalls
	snapshot := append([]domain.InspectionTask(nil), r.tasks...)
	r.mu.Unlock()
	switch call {
	case 1:
		close(r.firstListed)
		<-r.allowFirst
	case 2:
		close(r.secondListed)
	}
	return snapshot, nil
}

func (r *repository) SaveBundle(context.Context, store.TaskBundle, domain.Event, string, []byte) ([]byte, error) {
	return nil, domain.ErrNotFound
}
func (r *repository) Events(context.Context, string) ([]domain.Event, error) { return nil, nil }
func (r *repository) Idempotent(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}
func (r *repository) Check(context.Context) error { return nil }
func (r *repository) Close() error                { return nil }

func TestConcurrentCreateTaskRejectsOverlappingWindow(t *testing.T) {
	repo := newRepository()
	svc := application.New(repo)
	start := time.Date(2026, 8, 26, 1, 0, 0, 0, time.UTC)
	request := application.CreateTaskRequest{
		WindFarm: "并发风场", TurbineCode: "T-CONCURRENT", BladeCount: 3,
		InspectionWindowStart: start, InspectionWindowEnd: start.Add(2 * time.Hour), CreatedBy: "建案员",
	}
	results := make(chan error, 2)
	go func() { _, err := svc.CreateTask(context.Background(), request); results <- err }()
	select {
	case <-repo.firstListed:
	case <-time.After(2 * time.Second):
		t.Fatal("首个建案请求未完成列表读取")
	}
	go func() { _, err := svc.CreateTask(context.Background(), request); results <- err }()
	concurrent := false
	select {
	case <-repo.secondListed:
		concurrent = true
	case <-time.After(2 * time.Second):
	}
	close(repo.allowFirst)
	firstErr, secondErr := <-results, <-results
	if !concurrent {
		successes := 0
		for _, err := range []error{firstErr, secondErr} {
			if err == nil {
				successes++
			}
		}
		if successes != 1 {
			t.Fatalf("串行建案应接受首个任务并拒绝重叠任务，errors=%v,%v", firstErr, secondErr)
		}
		return
	}
	if firstErr == nil && secondErr == nil {
		t.Fatalf("并发建案绕过重叠窗口检查，两个请求均成功: tasks=%d", len(repo.tasks))
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("并发建案应只持久化一个任务，实际=%d", len(repo.tasks))
	}
}

package store

import (
	"bladeready/internal/domain"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteCreateLoadAuditAndIdempotency(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	task, err := domain.NewTask("task-1", "风场", "T-1", 1, now, now.Add(time.Hour), "甲")
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewEvent("event-1", task.ID, "task.created", "甲", 1, task)
	if err != nil {
		t.Fatal(err)
	}
	bundle := TaskBundle{Task: *task, Zones: []domain.BladeZone{}, Observations: []domain.DroneObservation{}, RepairPlan: []domain.RepairAction{}, Retests: []domain.RetestReading{}, Deviations: []domain.Deviation{}}
	raw, _ := json.Marshal(bundle)
	if _, err = repo.CreateTask(ctx, *task, event, "key-1", raw); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.LoadTask(ctx, task.ID)
	if err != nil || loaded.Task.Version != 1 {
		t.Fatalf("load: %+v %v", loaded, err)
	}
	cached, ok, err := repo.Idempotent(ctx, "key-1")
	if err != nil || !ok || len(cached) == 0 {
		t.Fatalf("idempotency: %t %v", ok, err)
	}
	events, err := repo.Events(ctx, task.ID)
	if err != nil || len(events) != 1 {
		t.Fatalf("events: %+v %v", events, err)
	}
	recovery, err := repo.Recover(ctx)
	if err != nil || recovery.IncompleteTasks != 1 {
		t.Fatalf("recovery: %+v %v", recovery, err)
	}
}

package idempotencykeycollision

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestIdempotencyKeyCannotReturnAnotherTasksBundle(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "collision.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := application.New(repo)
	now := time.Now().UTC()
	first, err := svc.CreateTask(context.Background(), application.CreateTaskRequest{
		WindFarm: "风场", TurbineCode: "T-COLLIDE-A", BladeCount: 1,
		InspectionWindowStart: now, InspectionWindowEnd: now.Add(time.Hour), CreatedBy: "建案员", IdempotencyKey: "shared-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateTask(context.Background(), application.CreateTaskRequest{
		WindFarm: "风场", TurbineCode: "T-COLLIDE-B", BladeCount: 1,
		InspectionWindowStart: now.Add(2 * time.Hour), InspectionWindowEnd: now.Add(3 * time.Hour), CreatedBy: "建案员",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.SetZones(context.Background(), second.Task.ID, application.SetZonesRequest{
		ExpectedVersion: second.Task.Version, Actor: "边界员", IdempotencyKey: "shared-key",
		Zones: []domain.BladeZone{{BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Task.ID != second.Task.ID {
		t.Fatalf("idempotency key returned task %s instead of target %s", updated.Task.ID, second.Task.ID)
	}
	persisted, err := svc.Get(context.Background(), second.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Task.Status != domain.StatusObserving || len(persisted.Zones) != 1 {
		t.Fatalf("target task was not persisted: status=%s zones=%d (first task=%s)", persisted.Task.Status, len(persisted.Zones), first.Task.ID)
	}
}

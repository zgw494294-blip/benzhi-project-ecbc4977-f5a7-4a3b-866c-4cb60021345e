package idempotencyreplaystaleversion

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestIdempotencyReplayWithStaleVersionReturnsCachedSuccess(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "replay.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := application.New(repo)
	now := time.Now().UTC()
	b, err := svc.CreateTask(context.Background(), application.CreateTaskRequest{
		WindFarm: "风场", TurbineCode: "T-REPLAY", BladeCount: 1,
		InspectionWindowStart: now, InspectionWindowEnd: now.Add(time.Hour), CreatedBy: "建案员",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := application.SetZonesRequest{ExpectedVersion: b.Task.Version, Actor: "边界员", IdempotencyKey: "zones-replay", Zones: []domain.BladeZone{{
		BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14,
	}}}
	first, err := svc.SetZones(context.Background(), b.Task.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	request.ExpectedVersion = 1
	second, err := svc.SetZones(context.Background(), b.Task.ID, request)
	if err != nil {
		if errors.Is(err, domain.ErrVersionConflict) {
			t.Fatalf("cached idempotent replay returned version conflict: %v", err)
		}
		t.Fatal(err)
	}
	if second.Task.Version != first.Task.Version || second.Task.Status != domain.StatusObserving {
		t.Fatalf("replay changed result: first=%+v second=%+v", first.Task, second.Task)
	}
}

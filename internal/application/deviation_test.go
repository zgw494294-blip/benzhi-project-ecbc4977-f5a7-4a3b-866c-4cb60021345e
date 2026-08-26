package application

import (
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestDeviationRequiresDirectedPassingRetest(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "deviation.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := New(repo)
	ctx := context.Background()
	now := time.Now().UTC()
	b, err := svc.CreateTask(ctx, CreateTaskRequest{WindFarm: "风场", TurbineCode: "T-2", BladeCount: 1, InspectionWindowStart: now, InspectionWindowEnd: now.Add(time.Hour), CreatedBy: "甲"})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.SetZones(ctx, b.Task.ID, SetZonesRequest{ExpectedVersion: b.Task.Version, Actor: "甲", Zones: []domain.BladeZone{{BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 5, MaxDelaminationMM: 10, OperatingWindLimit: 12}}})
	if err != nil {
		t.Fatal(err)
	}
	zone := b.Zones[0]
	b, err = svc.AddObservation(ctx, b.Task.ID, AddObservationRequest{ExpectedVersion: b.Task.Version, Actor: "甲", Observation: domain.DroneObservation{BladeZoneID: zone.ID, CapturedAt: now, DefectType: domain.DefectCrack, PositionMM: 100, LengthMM: 10, WidthMM: 2, PhotoDigest: "photo-test", Sequence: 1}})
	if err != nil {
		t.Fatal(err)
	}
	obs := b.Observations[0]
	b, err = svc.Assess(ctx, b.Task.ID, AssessRequest{ExpectedVersion: b.Task.Version, Actor: "甲", PlannedWind: 10})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.FreezeRepairPlan(ctx, b.Task.ID, FreezeRepairPlanRequest{ExpectedVersion: b.Task.Version, Actor: "乙", Actions: []domain.RepairAction{{ObservationID: obs.ID, Action: "补强", Owner: "乙", RetestPoint: "中心"}}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.AddRetests(ctx, b.Task.ID, AddRetestsRequest{ExpectedVersion: b.Task.Version, Actor: "丙", Readings: []domain.RetestReading{{ObservationID: obs.ID, BladeZoneID: zone.ID, MeasuredAt: now.Add(time.Hour), LengthMM: 5, WidthMM: 1, EvidenceDigest: "bad-evidence", Operator: "丙"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Deviations) != 1 {
		t.Fatalf("expected deviation: %+v", b.Deviations)
	}
	deviation := b.Deviations[0]
	_, err = svc.CloseDeviation(ctx, b.Task.ID, deviation.ID, CloseDeviationRequest{ExpectedVersion: b.Task.Version, Actor: "丁", CorrectiveAction: "尝试关闭"})
	if err == nil {
		t.Fatal("unsafe deviation close accepted")
	}
	b, err = svc.AddRetests(ctx, b.Task.ID, AddRetestsRequest{ExpectedVersion: b.Task.Version, Actor: "丙", Readings: []domain.RetestReading{{ObservationID: obs.ID, BladeZoneID: zone.ID, MeasuredAt: now.Add(2 * time.Hour), LengthMM: 1, WidthMM: .1, EvidenceDigest: "good-evidence", Operator: "丙"}}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.CloseDeviation(ctx, b.Task.ID, deviation.ID, CloseDeviationRequest{ExpectedVersion: b.Task.Version, Actor: "丁", CorrectiveAction: "再次补强且定向复测通过"})
	if err != nil {
		t.Fatal(err)
	}
	if b.Deviations[0].Status != "closed" {
		t.Fatal("deviation remained open")
	}
}

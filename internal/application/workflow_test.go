package application

import (
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestCompleteWorkflowAndVersionConflict(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "flow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := New(repo)
	ctx := context.Background()
	now := time.Now().UTC()
	b, err := svc.CreateTask(ctx, CreateTaskRequest{WindFarm: "测试风场", TurbineCode: "T-9", BladeCount: 1, InspectionWindowStart: now, InspectionWindowEnd: now.Add(time.Hour), CreatedBy: "建案员", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SetZones(ctx, b.Task.ID, SetZonesRequest{ExpectedVersion: 99, Actor: "甲", Zones: []domain.BladeZone{{BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14}}})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected version conflict: %v", err)
	}
	b, err = svc.SetZones(ctx, b.Task.ID, SetZonesRequest{ExpectedVersion: b.Task.Version, Actor: "甲", IdempotencyKey: "zones", Zones: []domain.BladeZone{{BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14}}})
	if err != nil {
		t.Fatal(err)
	}
	zone := b.Zones[0]
	b, err = svc.AddObservation(ctx, b.Task.ID, AddObservationRequest{ExpectedVersion: b.Task.Version, Actor: "乙", IdempotencyKey: "obs", Observation: domain.DroneObservation{BladeZoneID: zone.ID, CapturedAt: now, DefectType: domain.DefectCrack, PositionMM: 1000, LengthMM: 6, WidthMM: 2, PhotoDigest: "photo1234", Sequence: 1}})
	if err != nil {
		t.Fatal(err)
	}
	obs := b.Observations[0]
	b, err = svc.Assess(ctx, b.Task.ID, AssessRequest{ExpectedVersion: b.Task.Version, Actor: "丙", IdempotencyKey: "assess", PlannedWind: 10})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.FreezeRepairPlan(ctx, b.Task.ID, FreezeRepairPlanRequest{ExpectedVersion: b.Task.Version, Actor: "丁", IdempotencyKey: "plan", Actions: []domain.RepairAction{{ObservationID: obs.ID, Action: "补强", Owner: "维修组", RetestPoint: "中心点"}}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.AddRetests(ctx, b.Task.ID, AddRetestsRequest{ExpectedVersion: b.Task.Version, Actor: "戊", IdempotencyKey: "retest", Readings: []domain.RetestReading{{ObservationID: obs.ID, BladeZoneID: zone.ID, MeasuredAt: now.Add(time.Hour), LengthMM: 1, WidthMM: .2, EvidenceDigest: "evidence1234", Operator: "复测员"}}})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.PrepareReview(ctx, b.Task.ID, PrepareReviewRequest{ExpectedVersion: b.Task.Version, Actor: "己", IdempotencyKey: "review"})
	if err != nil {
		t.Fatal(err)
	}
	b, err = svc.Release(ctx, b.Task.ID, ReleaseRequest{ExpectedVersion: b.Task.Version, Reviewer: "安全员", IdempotencyKey: "release", ConfirmBoundary: true, ConfirmRetests: true, ConfirmAudit: true})
	if err != nil {
		t.Fatal(err)
	}
	if b.Task.Status != domain.StatusReleased || b.Credential == nil || len(b.Credential.CredentialDigest) != 64 {
		t.Fatalf("not released: %+v", b)
	}
	events, err := svc.Audit(ctx, b.Task.ID)
	if err != nil || len(events) != 8 {
		t.Fatalf("audit events=%d err=%v", len(events), err)
	}
}

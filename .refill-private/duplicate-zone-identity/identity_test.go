package duplicatezoneidentity

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSetZonesRejectsDuplicateZoneIDsBeforeAssessmentCanOverwriteBoundary(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := application.New(repo)
	now := time.Now().UTC()
	b, err := svc.CreateTask(context.Background(), application.CreateTaskRequest{
		WindFarm: "风场", TurbineCode: "T-ID", BladeCount: 1,
		InspectionWindowStart: now, InspectionWindowEnd: now.Add(time.Hour), CreatedBy: "建案员",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SetZones(context.Background(), b.Task.ID, application.SetZonesRequest{
		ExpectedVersion: b.Task.Version, Actor: "边界员", Zones: []domain.BladeZone{
			{ID: "zone-shared", BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 100, MaxDelaminationMM: 100, OperatingWindLimit: 14},
			{ID: "zone-shared", BladeIndex: 1, ZoneCode: "TIP", MaterialType: "玻纤", MaxCrackMM: 1, MaxDelaminationMM: 1, OperatingWindLimit: 14},
		},
	})
	if err == nil {
		t.Fatalf("duplicate zone identity was accepted; assessment indexes zones by ID and will overwrite one boundary")
	}
}

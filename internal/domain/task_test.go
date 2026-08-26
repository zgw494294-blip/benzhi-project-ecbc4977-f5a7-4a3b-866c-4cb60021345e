package domain

import (
	"errors"
	"testing"
	"time"
)

func TestTaskStateAndVersion(t *testing.T) {
	start := time.Now().UTC()
	task, err := NewTask("t1", "测试风场", "T-01", 3, start, start.Add(time.Hour), "工程师")
	if err != nil {
		t.Fatal(err)
	}
	if err = task.Move(99, StatusObserving); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err = task.Move(1, StatusRepairing); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected transition error, got %v", err)
	}
	for _, status := range []TaskStatus{StatusObserving, StatusAssessed, StatusRepairing, StatusRetesting, StatusReviewing, StatusReleased} {
		if err = task.Move(task.Version, status); err != nil {
			t.Fatalf("move to %s: %v", status, err)
		}
	}
	if task.ClosedAt == nil {
		t.Fatal("released task must have closed_at")
	}
	if err = task.Move(task.Version, StatusRetesting); !errors.Is(err, ErrCredentialIssued) {
		t.Fatalf("released mutation should fail: %v", err)
	}
}

func TestBoundaryAndTimelinePolicy(t *testing.T) {
	zones := []BladeZone{{BladeIndex: 1, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14}}
	if err := ValidateBoundarySet(2, zones); err == nil {
		t.Fatal("partial blade boundary accepted")
	}
	zones = append(zones, BladeZone{BladeIndex: 2, ZoneCode: "ROOT", MaterialType: "玻纤", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 14})
	if err := ValidateBoundarySet(2, zones); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	observations := []DroneObservation{{ID: "o", BladeZoneID: "z", CapturedAt: now, DefectType: DefectCrack, LengthMM: 3, PhotoDigest: "12345678", Sequence: 2}}
	if err := ValidateObservationTimeline(observations); err == nil {
		t.Fatal("gap in sequence accepted")
	}
}

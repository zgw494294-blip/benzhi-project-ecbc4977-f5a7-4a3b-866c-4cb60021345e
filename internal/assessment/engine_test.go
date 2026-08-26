package assessment

import (
	"bladeready/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestRiskExplanationAndRetestDeviation(t *testing.T) {
	zone := domain.BladeZone{ID: "z1", ZoneCode: "ROOT-A", MaxCrackMM: 10, MaxDelaminationMM: 20, OperatingWindLimit: 12}
	obs := domain.DroneObservation{ID: "o1", BladeZoneID: "z1", CapturedAt: time.Now(), DefectType: domain.DefectCrack, PositionMM: 1000, LengthMM: 14, WidthMM: 2, PhotoDigest: "12345678", Sequence: 1}
	result := NewEngine().Evaluate(zone, obs, 15)
	if result.Level != "critical" || !result.Blocked {
		t.Fatalf("unexpected risk: %+v", result)
	}
	snapshot := domain.AssessmentSnapshot{HighestLevel: result.Level, Results: []domain.RiskResult{result}}
	text := Explain(snapshot).Text()
	if !strings.Contains(text, "阻断观测 1 项") {
		t.Fatalf("bad explanation %q", text)
	}
	deviations := NewComparator().Compare("t1", []domain.DroneObservation{obs}, nil)
	if len(deviations) != 1 || deviations[0].Kind != "missing" {
		t.Fatalf("missing retest not found: %+v", deviations)
	}
	retests := []domain.RetestReading{{ObservationID: "o1", BladeZoneID: "z1", LengthMM: 13, WidthMM: 2}}
	deviations = NewComparator().Compare("t1", []domain.DroneObservation{obs}, retests)
	if len(deviations) != 1 || deviations[0].Kind != "over_limit" {
		t.Fatalf("residual defect not found: %+v", deviations)
	}
}

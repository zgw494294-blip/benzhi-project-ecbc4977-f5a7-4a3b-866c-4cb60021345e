package assessment

import (
	"bladeready/internal/domain"
	"fmt"
)

type Comparator struct{}

func NewComparator() *Comparator { return &Comparator{} }

func (c *Comparator) Compare(taskID string, originals []domain.DroneObservation, retests []domain.RetestReading) []domain.Deviation {
	retestByObservation := make(map[string]domain.RetestReading)
	for _, r := range retests {
		if current, ok := retestByObservation[r.ObservationID]; !ok || r.MeasuredAt.After(current.MeasuredAt) {
			retestByObservation[r.ObservationID] = r
		}
	}
	var deviations []domain.Deviation
	for _, original := range originals {
		r, ok := retestByObservation[original.ID]
		if !ok {
			deviations = append(deviations, domain.Deviation{TaskID: taskID, ObservationID: original.ID, BladeZoneID: original.BladeZoneID,
				Kind: "missing", Reason: "缺少对应复测读数", Status: "open"})
			continue
		}
		if r.BladeZoneID != original.BladeZoneID {
			deviations = append(deviations, domain.Deviation{TaskID: taskID, ObservationID: original.ID, BladeZoneID: original.BladeZoneID,
				Kind: "wrong_zone", Reason: "复测分区与原缺陷不一致", Status: "open"})
			continue
		}
		if r.LengthMM > original.LengthMM || r.WidthMM > original.WidthMM {
			deviations = append(deviations, domain.Deviation{TaskID: taskID, ObservationID: original.ID, BladeZoneID: original.BladeZoneID,
				Kind: "regression", Reason: fmt.Sprintf("复测尺寸 %.1fx%.1f 大于原始 %.1fx%.1f", r.LengthMM, r.WidthMM, original.LengthMM, original.WidthMM), Status: "open"})
		} else if r.LengthMM > original.LengthMM*.2 {
			deviations = append(deviations, domain.Deviation{TaskID: taskID, ObservationID: original.ID, BladeZoneID: original.BladeZoneID,
				Kind: "over_limit", Reason: "维修后残余尺寸超过原始缺陷的 20%", Status: "open"})
		}
	}
	return deviations
}

package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// BoundaryDigest produces a stable fingerprint independent of submission order.
func BoundaryDigest(zones []BladeZone) string {
	ordered := append([]BladeZone(nil), zones...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].BladeIndex != ordered[j].BladeIndex {
			return ordered[i].BladeIndex < ordered[j].BladeIndex
		}
		return ordered[i].ZoneCode < ordered[j].ZoneCode
	})
	type boundary struct {
		BladeIndex         int     `json:"blade_index"`
		ZoneCode           string  `json:"zone_code"`
		MaterialType       string  `json:"material_type"`
		MaxCrackMM         float64 `json:"max_crack_mm"`
		MaxDelaminationMM  float64 `json:"max_delamination_mm"`
		OperatingWindLimit float64 `json:"operating_wind_limit"`
	}
	items := make([]boundary, 0, len(ordered))
	for _, z := range ordered {
		items = append(items, boundary{z.BladeIndex, z.ZoneCode, z.MaterialType, z.MaxCrackMM, z.MaxDelaminationMM, z.OperatingWindLimit})
	}
	raw, _ := json.Marshal(items)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func BoundaryCoverage(bladeCount int, zones []BladeZone) map[int]int {
	coverage := make(map[int]int, bladeCount)
	for _, z := range zones {
		coverage[z.BladeIndex]++
	}
	return coverage
}

func ValidateBoundaryIntegrity(bladeCount int, zones []BladeZone, digest string, coverage map[int]int) error {
	if digest == "" || digest != BoundaryDigest(zones) {
		return ValidationError{"boundary_digest", "冻结边界摘要不一致"}
	}
	if err := ValidateBoundarySet(bladeCount, zones); err != nil {
		return ValidationError{"boundary", "冻结边界覆盖不完整: " + err.Error()}
	}
	expected := BoundaryCoverage(bladeCount, zones)
	for blade, count := range expected {
		if coverage == nil || coverage[blade] != count {
			return ValidationError{"boundary_coverage", fmt.Sprintf("第 %d 片叶片分区覆盖索引不一致", blade)}
		}
	}
	return nil
}

// ValidateBoundarySet verifies that every configured blade is represented and
// that a zone key is unique within a blade. A task may contain several zones
// per blade, but it cannot freeze a partial blade set.
func ValidateBoundarySet(bladeCount int, zones []BladeZone) error {
	if len(zones) < bladeCount {
		return ValidationError{"zones", "冻结前必须覆盖任务中的每片叶片"}
	}
	blades := make(map[int]bool)
	keys := make(map[string]bool)
	ids := make(map[string]bool)
	for _, z := range zones {
		if err := z.Validate(bladeCount); err != nil {
			return err
		}
		key := fmt.Sprintf("%d/%s", z.BladeIndex, z.ZoneCode)
		if keys[key] {
			return ValidationError{"zones", "同一叶片的分区代码不能重复"}
		}
		keys[key] = true
		// A frozen boundary must reference each zone by a stable, unambiguous
		// identifier so observations cannot resolve to the wrong thresholds.
		// Empty identifiers are assigned later by the registration flow and are
		// therefore excluded from the uniqueness check.
		if z.ID != "" {
			if ids[z.ID] {
				return ValidationError{"zones", "分区标识符不能重复"}
			}
			ids[z.ID] = true
		}
		blades[z.BladeIndex] = true
	}
	for blade := 1; blade <= bladeCount; blade++ {
		if !blades[blade] {
			return ValidationError{"zones", fmt.Sprintf("第 %d 片叶片没有运行边界", blade)}
		}
	}
	return nil
}

// ValidateObservationTimeline guarantees a stable, gap-free evidence order.
// This makes sequence meaningful when an audit is replayed.
func ValidateObservationTimeline(observations []DroneObservation) error {
	if len(observations) == 0 {
		return ValidationError{"observations", "没有观测证据"}
	}
	sequences := make([]int, 0, len(observations))
	seen := make(map[int]bool)
	for _, observation := range observations {
		if err := observation.Validate(); err != nil {
			return err
		}
		if seen[observation.Sequence] {
			return ValidationError{"sequence", "观测序号重复"}
		}
		seen[observation.Sequence] = true
		sequences = append(sequences, observation.Sequence)
	}
	sort.Ints(sequences)
	for i, sequence := range sequences {
		if sequence != i+1 {
			return ValidationError{"sequence", "观测序号必须连续且从 1 开始"}
		}
	}
	return nil
}

func ValidateRepairCoverage(results []RiskResult, actions []RepairAction) error {
	required := make(map[string]bool)
	for _, result := range results {
		if result.Blocked {
			required[result.ObservationID] = true
		}
	}
	covered := make(map[string]bool)
	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return err
		}
		if covered[action.ObservationID] {
			return ValidationError{"actions", "同一观测存在重复维修动作"}
		}
		covered[action.ObservationID] = true
	}
	for observationID := range required {
		if !covered[observationID] {
			return ValidationError{"actions", "高风险观测 " + observationID + " 缺少维修动作"}
		}
	}
	return nil
}

func ValidateReviewEvidence(retests []RetestReading, deviations []Deviation) error {
	if len(retests) == 0 {
		return ValidationError{"retests", "缺少维修后复测证据"}
	}
	for _, reading := range retests {
		if err := reading.Validate(); err != nil {
			return err
		}
	}
	for _, deviation := range deviations {
		if deviation.Status != "closed" {
			return ValidationError{"deviations", "仍有未关闭偏差 " + deviation.ID}
		}
		if deviation.CorrectiveAction == "" || deviation.ClosedBy == "" || deviation.ClosedAt == nil {
			return ValidationError{"deviations", "已关闭偏差缺少整改证据"}
		}
	}
	return nil
}

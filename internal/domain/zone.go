package domain

import "time"

type BladeZone struct {
	ID                 string     `json:"id"`
	TaskID             string     `json:"task_id"`
	BladeIndex         int        `json:"blade_index"`
	ZoneCode           string     `json:"zone_code"`
	MaterialType       string     `json:"material_type"`
	MaxCrackMM         float64    `json:"max_crack_mm"`
	MaxDelaminationMM  float64    `json:"max_delamination_mm"`
	OperatingWindLimit float64    `json:"operating_wind_limit"`
	FrozenAt           *time.Time `json:"frozen_at,omitempty"`
}

func (z BladeZone) Validate(bladeCount int) error {
	if z.BladeIndex < 1 || z.BladeIndex > bladeCount {
		return ValidationError{"blade_index", "超出任务叶片范围"}
	}
	if err := Required("zone_code", z.ZoneCode); err != nil {
		return err
	}
	if err := Required("material_type", z.MaterialType); err != nil {
		return err
	}
	if z.MaxCrackMM <= 0 {
		return ValidationError{"max_crack_mm", "必须大于 0"}
	}
	if z.MaxDelaminationMM <= 0 {
		return ValidationError{"max_delamination_mm", "必须大于 0"}
	}
	if z.OperatingWindLimit <= 0 {
		return ValidationError{"operating_wind_limit", "必须大于 0"}
	}
	return nil
}

func (z *BladeZone) Freeze(now time.Time) error {
	if z.FrozenAt != nil {
		return ErrFrozen
	}
	z.FrozenAt = &now
	return nil
}

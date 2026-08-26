package domain

import "time"

type DefectType string

const (
	DefectCrack        DefectType = "crack"
	DefectDelamination DefectType = "delamination"
	DefectErosion      DefectType = "erosion"
	DefectLightning    DefectType = "lightning"
)

type DroneObservation struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	BladeZoneID string     `json:"blade_zone_id"`
	CapturedAt  time.Time  `json:"captured_at"`
	DefectType  DefectType `json:"defect_type"`
	PositionMM  float64    `json:"position_mm"`
	LengthMM    float64    `json:"length_mm"`
	WidthMM     float64    `json:"width_mm"`
	PhotoDigest string     `json:"photo_digest"`
	ReviewNote  string     `json:"review_note"`
	Sequence    int        `json:"sequence"`
}

func (o DroneObservation) Validate() error {
	if err := Required("blade_zone_id", o.BladeZoneID); err != nil {
		return err
	}
	if o.CapturedAt.IsZero() {
		return ValidationError{"captured_at", "不能为空"}
	}
	switch o.DefectType {
	case DefectCrack, DefectDelamination, DefectErosion, DefectLightning:
	default:
		return ValidationError{"defect_type", "不支持的缺陷类型"}
	}
	if o.PositionMM < 0 || o.LengthMM <= 0 || o.WidthMM < 0 {
		return ValidationError{"measurement", "位置和尺寸不合法"}
	}
	if len(o.PhotoDigest) < 8 {
		return ValidationError{"photo_digest", "照片摘要至少 8 个字符"}
	}
	if o.Sequence < 1 {
		return ValidationError{"sequence", "必须从 1 开始"}
	}
	return nil
}

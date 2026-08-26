package domain

import "time"

type RepairAction struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	ObservationID string     `json:"observation_id"`
	Action        string     `json:"action"`
	Owner         string     `json:"owner"`
	RetestPoint   string     `json:"retest_point"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

func (r RepairAction) Validate() error {
	if err := Required("observation_id", r.ObservationID); err != nil {
		return err
	}
	if err := Required("action", r.Action); err != nil {
		return err
	}
	if err := Required("owner", r.Owner); err != nil {
		return err
	}
	return Required("retest_point", r.RetestPoint)
}

type RetestReading struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"task_id"`
	ObservationID  string    `json:"observation_id"`
	BladeZoneID    string    `json:"blade_zone_id"`
	MeasuredAt     time.Time `json:"measured_at"`
	LengthMM       float64   `json:"length_mm"`
	WidthMM        float64   `json:"width_mm"`
	EvidenceDigest string    `json:"evidence_digest"`
	Operator       string    `json:"operator"`
}

func (r RetestReading) Validate() error {
	if err := Required("observation_id", r.ObservationID); err != nil {
		return err
	}
	if err := Required("blade_zone_id", r.BladeZoneID); err != nil {
		return err
	}
	if r.MeasuredAt.IsZero() {
		return ValidationError{"measured_at", "不能为空"}
	}
	if r.LengthMM < 0 || r.WidthMM < 0 {
		return ValidationError{"measurement", "不能为负数"}
	}
	if len(r.EvidenceDigest) < 8 {
		return ValidationError{"evidence_digest", "证据摘要至少 8 个字符"}
	}
	return Required("operator", r.Operator)
}

type Deviation struct {
	ID               string     `json:"id"`
	TaskID           string     `json:"task_id"`
	ObservationID    string     `json:"observation_id"`
	BladeZoneID      string     `json:"blade_zone_id"`
	Kind             string     `json:"kind"`
	Reason           string     `json:"reason"`
	Status           string     `json:"status"`
	CorrectiveAction string     `json:"corrective_action,omitempty"`
	ClosedBy         string     `json:"closed_by,omitempty"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
}

func (d *Deviation) Close(action, reviewer string, now time.Time) error {
	if d.Status == "closed" {
		return ValidationError{"deviation", "偏差已经关闭"}
	}
	if err := Required("corrective_action", action); err != nil {
		return err
	}
	if err := Required("closed_by", reviewer); err != nil {
		return err
	}
	d.CorrectiveAction, d.ClosedBy, d.Status, d.ClosedAt = action, reviewer, "closed", &now
	return nil
}

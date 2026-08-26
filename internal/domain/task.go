package domain

import (
	"fmt"
	"time"
)

type InspectionTask struct {
	ID                    string     `json:"id"`
	WindFarm              string     `json:"wind_farm"`
	TurbineCode           string     `json:"turbine_code"`
	BladeCount            int        `json:"blade_count"`
	InspectionWindowStart time.Time  `json:"inspection_window_start"`
	InspectionWindowEnd   time.Time  `json:"inspection_window_end"`
	Status                TaskStatus `json:"status"`
	Version               int64      `json:"version"`
	CreatedBy             string     `json:"created_by"`
	CreatedAt             time.Time  `json:"created_at"`
	ClosedAt              *time.Time `json:"closed_at,omitempty"`
	Overdue               bool       `json:"overdue,omitempty"`
}

func NewTask(id, farm, turbine string, blades int, start, end time.Time, creator string) (*InspectionTask, error) {
	if err := Required("wind_farm", farm); err != nil {
		return nil, err
	}
	if err := Required("turbine_code", turbine); err != nil {
		return nil, err
	}
	if err := Required("created_by", creator); err != nil {
		return nil, err
	}
	if blades < 1 || blades > 9 {
		return nil, ValidationError{"blade_count", "必须在 1 到 9 之间"}
	}
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return nil, ValidationError{"inspection_window", "结束时间必须晚于开始时间"}
	}
	return &InspectionTask{ID: id, WindFarm: farm, TurbineCode: turbine, BladeCount: blades,
		InspectionWindowStart: start.UTC(), InspectionWindowEnd: end.UTC(), Status: StatusDraft, Version: 1,
		CreatedBy: creator, CreatedAt: time.Now().UTC()}, nil
}

func (t *InspectionTask) CheckVersion(expected int64) error {
	if t.Version != expected {
		return fmt.Errorf("%w: 当前版本 %d，提交版本 %d", ErrVersionConflict, t.Version, expected)
	}
	return nil
}

func (t *InspectionTask) Move(expected int64, next TaskStatus) error {
	if err := t.CheckVersion(expected); err != nil {
		return err
	}
	if t.Status == StatusReleased {
		return ErrCredentialIssued
	}
	if !CanTransition(t.Status, next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, t.Status, next)
	}
	t.Status, t.Version = next, t.Version+1
	if next == StatusReleased {
		now := time.Now().UTC()
		t.ClosedAt = &now
	}
	return nil
}

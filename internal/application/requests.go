package application

import (
	"bladeready/internal/domain"
	"time"
)

type CreateTaskRequest struct {
	WindFarm              string    `json:"wind_farm"`
	TurbineCode           string    `json:"turbine_code"`
	BladeCount            int       `json:"blade_count"`
	InspectionWindowStart time.Time `json:"inspection_window_start"`
	InspectionWindowEnd   time.Time `json:"inspection_window_end"`
	CreatedBy             string    `json:"created_by"`
	IdempotencyKey        string    `json:"idempotency_key"`
}
type SetZonesRequest struct {
	ExpectedVersion int64              `json:"expected_version"`
	Actor           string             `json:"actor"`
	IdempotencyKey  string             `json:"idempotency_key"`
	Zones           []domain.BladeZone `json:"zones"`
}
type AddObservationRequest struct {
	ExpectedVersion int64                   `json:"expected_version"`
	Actor           string                  `json:"actor"`
	IdempotencyKey  string                  `json:"idempotency_key"`
	Observation     domain.DroneObservation `json:"observation"`
}
type AssessRequest struct {
	ExpectedVersion int64   `json:"expected_version"`
	Actor           string  `json:"actor"`
	IdempotencyKey  string  `json:"idempotency_key"`
	PlannedWind     float64 `json:"planned_wind"`
}
type FreezeRepairPlanRequest struct {
	ExpectedVersion int64                 `json:"expected_version"`
	Actor           string                `json:"actor"`
	IdempotencyKey  string                `json:"idempotency_key"`
	Actions         []domain.RepairAction `json:"actions"`
}
type AddRetestsRequest struct {
	ExpectedVersion int64                  `json:"expected_version"`
	Actor           string                 `json:"actor"`
	IdempotencyKey  string                 `json:"idempotency_key"`
	Readings        []domain.RetestReading `json:"readings"`
}
type CloseDeviationRequest struct {
	ExpectedVersion  int64  `json:"expected_version"`
	Actor            string `json:"actor"`
	IdempotencyKey   string `json:"idempotency_key"`
	CorrectiveAction string `json:"corrective_action"`
}
type PrepareReviewRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	Actor           string `json:"actor"`
	IdempotencyKey  string `json:"idempotency_key"`
}
type ReleaseRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reviewer        string `json:"reviewer"`
	IdempotencyKey  string `json:"idempotency_key"`
	ConfirmBoundary bool   `json:"confirm_boundary"`
	ConfirmRetests  bool   `json:"confirm_retests"`
	ConfirmAudit    bool   `json:"confirm_audit"`
}

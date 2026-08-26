package store

import (
	"bladeready/internal/domain"
	"context"
	"time"
)

type TaskBundle struct {
	Task             domain.InspectionTask      `json:"task"`
	Zones            []domain.BladeZone         `json:"zones"`
	Observations     []domain.DroneObservation  `json:"observations"`
	Assessment       *domain.AssessmentSnapshot `json:"assessment,omitempty"`
	RepairPlan       []domain.RepairAction      `json:"repair_plan"`
	Retests          []domain.RetestReading     `json:"retests"`
	Deviations       []domain.Deviation         `json:"deviations"`
	Credential       *domain.ReleaseCredential  `json:"credential,omitempty"`
	BoundaryDigest   string                     `json:"boundary_digest,omitempty"`
	BoundarySummary  string                     `json:"boundary_summary,omitempty"`
	BoundaryFrozenAt *time.Time                 `json:"boundary_frozen_at,omitempty"`
	ZoneCoverage     map[int]int                `json:"zone_coverage,omitempty"`
}

type TaskListFilter struct {
	Status   domain.TaskStatus
	WindFarm string
	From, To *time.Time
}

type Repository interface {
	CreateTask(context.Context, domain.InspectionTask, domain.Event, string, []byte) ([]byte, error)
	LoadTask(context.Context, string) (TaskBundle, error)
	ListTasks(context.Context) ([]domain.InspectionTask, error)
	SaveBundle(context.Context, TaskBundle, domain.Event, string, []byte) ([]byte, error)
	Events(context.Context, string) ([]domain.Event, error)
	Idempotent(context.Context, string) ([]byte, bool, error)
	Check(context.Context) error
	Close() error
}

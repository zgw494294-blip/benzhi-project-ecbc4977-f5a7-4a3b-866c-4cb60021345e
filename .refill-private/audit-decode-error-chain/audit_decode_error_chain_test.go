package auditdecodeerrorchain_test

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestAuditReplayPreservesDecodeErrorChain(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 26, 8, 0, 0, 0, time.UTC)
	task, err := domain.NewTask("task-audit-chain", "北岭风场", "T-17", 1, now, now.Add(time.Hour), "巡检员")
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewEvent("event-audit-chain", task.ID, "task.created", "巡检员", task.Version, task)
	if err != nil {
		t.Fatal(err)
	}
	bundle := store.TaskBundle{
		Task:         *task,
		Zones:        []domain.BladeZone{},
		Observations: []domain.DroneObservation{},
		RepairPlan:   []domain.RepairAction{},
		Retests:      []domain.RetestReading{},
		Deviations:   []domain.Deviation{},
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.CreateTask(context.Background(), *task, event, "audit-chain-create", raw); err != nil {
		t.Fatal(err)
	}
	if err = repo.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE audit_events SET occurred_at=? WHERE id=?", "not-rfc3339", event.ID); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}

	repo, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	_, err = application.New(repo).Audit(context.Background(), task.ID)
	if err == nil {
		t.Fatal("损坏的 occurred_at 未导致审计重放失败")
	}
	var parseErr *time.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("审计解码错误链丢失，errors.As 无法识别 time.ParseError: %v", err)
	}
}

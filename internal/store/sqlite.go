package store

import (
	"bladeready/internal/domain"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"time"
)

type SQLite struct{ db *sql.DB }

func Open(path string) (*SQLite, error) {
	if path == "" {
		path = "bladeready.db"
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	s := &SQLite{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("执行迁移: %w", err)
	}
	if err = s.Check(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if _, err = s.Recover(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }
func (s *SQLite) Check(ctx context.Context) error {
	var result string
	if err := s.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("数据库完整性检查失败: %s", result)
	}
	return nil
}

func (s *SQLite) Idempotent(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	var response []byte
	err := s.db.QueryRowContext(ctx, "SELECT response FROM idempotency WHERE key=?", key).Scan(&response)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return response, err == nil, err
}

func (s *SQLite) CreateTask(ctx context.Context, task domain.InspectionTask, event domain.Event, key string, response []byte) ([]byte, error) {
	bundle := TaskBundle{Task: task, Zones: []domain.BladeZone{}, Observations: []domain.DroneObservation{}, RepairPlan: []domain.RepairAction{}, Retests: []domain.RetestReading{}, Deviations: []domain.Deviation{}}
	if cached, ok, err := s.Idempotent(ctx, key); err != nil || ok {
		return cached, err
	}
	b, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO tasks(id,wind_farm,turbine_code,status,version,snapshot,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		task.ID, task.WindFarm, task.TurbineCode, task.Status, task.Version, b, task.CreatedAt.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	if err = insertEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if key != "" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO idempotency(key,response,created_at) VALUES(?,?,?)", key, response, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *SQLite) LoadTask(ctx context.Context, id string) (TaskBundle, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, "SELECT snapshot FROM tasks WHERE id=?", id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskBundle{}, domain.ErrNotFound
	}
	if err != nil {
		return TaskBundle{}, err
	}
	var bundle TaskBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return TaskBundle{}, fmt.Errorf("解析任务快照: %w", err)
	}
	return bundle, nil
}

func (s *SQLite) ListTasks(ctx context.Context) ([]domain.InspectionTask, error) {
	return s.ListTasksFiltered(ctx, TaskListFilter{})
}

func (s *SQLite) ListTasksFiltered(ctx context.Context, filter TaskListFilter) ([]domain.InspectionTask, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT snapshot FROM tasks ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.InspectionTask{}
	for rows.Next() {
		var raw []byte
		var b TaskBundle
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		if filter.Status != "" && b.Task.Status != filter.Status {
			continue
		}
		if filter.WindFarm != "" && b.Task.WindFarm != filter.WindFarm {
			continue
		}
		if filter.From != nil && b.Task.InspectionWindowEnd.Before(*filter.From) {
			continue
		}
		if filter.To != nil && b.Task.InspectionWindowStart.After(*filter.To) {
			continue
		}
		b.Task.Overdue = (b.Task.Status == domain.StatusDraft || b.Task.Status == domain.StatusObserving) && time.Now().UTC().After(b.Task.InspectionWindowEnd)
		result = append(result, b.Task)
	}
	return result, rows.Err()
}

func (s *SQLite) SaveBundle(ctx context.Context, bundle TaskBundle, event domain.Event, key string, response []byte) ([]byte, error) {
	if cached, ok, err := s.Idempotent(ctx, key); err != nil || ok {
		return cached, err
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE tasks SET status=?,version=?,snapshot=?,updated_at=? WHERE id=? AND version=?`,
		bundle.Task.Status, bundle.Task.Version, raw, time.Now().UTC().Format(time.RFC3339Nano), bundle.Task.ID, event.Version-1)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, domain.ErrVersionConflict
	}
	if err = syncObservations(ctx, tx, bundle); err != nil {
		return nil, err
	}
	if bundle.Credential != nil {
		if err = insertCredential(ctx, tx, *bundle.Credential); err != nil {
			return nil, err
		}
	}
	if err = insertEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if key != "" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO idempotency(key,response,created_at) VALUES(?,?,?)", key, response, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return response, nil
}

func insertEvent(ctx context.Context, tx *sql.Tx, e domain.Event) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,task_id,version,event_type,actor,occurred_at,payload) VALUES(?,?,?,?,?,?,?)`, e.ID, e.TaskID, e.Version, e.Type, e.Actor, e.At.Format(time.RFC3339Nano), []byte(e.Payload))
	return err
}
func syncObservations(ctx context.Context, tx *sql.Tx, b TaskBundle) error {
	for _, o := range b.Observations {
		raw, _ := json.Marshal(o)
		_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO observations(id,task_id,zone_id,sequence,captured_at,payload) VALUES(?,?,?,?,?,?)`, o.ID, o.TaskID, o.BladeZoneID, o.Sequence, o.CapturedAt.Format(time.RFC3339Nano), raw)
		if err != nil {
			return err
		}
	}
	return nil
}
func insertCredential(ctx context.Context, tx *sql.Tx, c domain.ReleaseCredential) error {
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM credentials WHERE task_id=?", c.TaskID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	raw, _ := json.Marshal(c)
	_, err := tx.ExecContext(ctx, `INSERT INTO credentials(id,task_id,digest,payload,issued_at) VALUES(?,?,?,?,?)`, c.ID, c.TaskID, c.CredentialDigest, raw, c.IssuedAt.Format(time.RFC3339Nano))
	return err
}

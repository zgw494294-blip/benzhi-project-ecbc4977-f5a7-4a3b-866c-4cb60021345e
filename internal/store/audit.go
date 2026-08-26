package store

import (
	"bladeready/internal/domain"
	"context"
	"time"
)

func (s *SQLite) Events(ctx context.Context, taskID string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,task_id,version,event_type,actor,occurred_at,payload FROM audit_events WHERE task_id=? ORDER BY occurred_at,id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.Event{}
	for rows.Next() {
		var e domain.Event
		var at string
		if err = rows.Scan(&e.ID, &e.TaskID, &e.Version, &e.Type, &e.Actor, &at, &e.Payload); err != nil {
			return nil, err
		}
		e.At, _ = time.Parse(time.RFC3339Nano, at)
		events = append(events, e)
	}
	return events, rows.Err()
}

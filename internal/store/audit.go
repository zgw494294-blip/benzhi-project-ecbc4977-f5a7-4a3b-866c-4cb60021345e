package store

import (
	"bladeready/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *SQLite) Events(ctx context.Context, taskID string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,task_id,version,event_type,actor,occurred_at,payload FROM audit_events WHERE task_id=? ORDER BY occurred_at,id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.Event{}
	decodeIssues := []string{}
	for rows.Next() {
		var e domain.Event
		var at string
		if err = rows.Scan(&e.ID, &e.TaskID, &e.Version, &e.Type, &e.Actor, &at, &e.Payload); err != nil {
			return nil, err
		}
		e.At, err = decodeAuditFields(at, e.Payload)
		if err != nil {
			decodeIssues = append(decodeIssues, fmt.Sprintf("事件 %s: %v", e.ID, err))
			continue
		}
		events = append(events, e)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(decodeIssues) > 0 {
		return nil, fmt.Errorf("审计记录解码失败: %s", strings.Join(decodeIssues, "; "))
	}
	return events, nil
}

func decodeAuditFields(at string, payload json.RawMessage) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return time.Time{}, fmt.Errorf("解析 occurred_at: %w", err)
	}
	if !json.Valid(payload) {
		return time.Time{}, fmt.Errorf("解析 payload: JSON 格式无效")
	}
	return parsed, nil
}

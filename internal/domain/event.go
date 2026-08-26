package domain

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID      string          `json:"id"`
	TaskID  string          `json:"task_id"`
	Version int64           `json:"version"`
	Type    string          `json:"type"`
	Actor   string          `json:"actor"`
	At      time.Time       `json:"at"`
	Payload json.RawMessage `json:"payload"`
}

func NewEvent(id, taskID, typ, actor string, version int64, value any) (Event, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return Event{}, err
	}
	return Event{ID: id, TaskID: taskID, Version: version, Type: typ, Actor: actor, At: time.Now().UTC(), Payload: b}, nil
}

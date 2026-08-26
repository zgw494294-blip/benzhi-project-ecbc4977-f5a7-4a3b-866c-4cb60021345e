package store

import (
	"bladeready/internal/domain"
	"context"
	"encoding/json"
	"fmt"
)

type RecoveryReport struct {
	IncompleteTasks int              `json:"incomplete_tasks"`
	ReleasedTasks   int              `json:"released_tasks"`
	LatestVersions  map[string]int64 `json:"latest_versions"`
}

// Recover scans durable snapshots during startup. A malformed or versionless
// task fails startup rather than allowing later writes over uncertain state.
func (s *SQLite) Recover(ctx context.Context) (RecoveryReport, error) {
	report := RecoveryReport{LatestVersions: make(map[string]int64)}
	rows, err := s.db.QueryContext(ctx, "SELECT id,status,version,snapshot FROM tasks ORDER BY id")
	if err != nil {
		return report, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, status string
		var version int64
		var raw []byte
		if err = rows.Scan(&id, &status, &version, &raw); err != nil {
			return report, err
		}
		var bundle TaskBundle
		if err = json.Unmarshal(raw, &bundle); err != nil {
			return report, fmt.Errorf("恢复任务 %s: %w", id, err)
		}
		if bundle.Task.ID != id || bundle.Task.Version != version || version < 1 {
			return report, fmt.Errorf("任务 %s 的快照身份或版本不一致", id)
		}
		if string(bundle.Task.Status) != status {
			return report, fmt.Errorf("任务 %s 的状态索引不一致", id)
		}
		report.LatestVersions[id] = version
		if bundle.Task.Status == domain.StatusReleased {
			if bundle.Credential == nil {
				return report, fmt.Errorf("已放行任务 %s 缺少凭证", id)
			}
			report.ReleasedTasks++
		} else {
			report.IncompleteTasks++
		}
	}
	return report, rows.Err()
}

package tasklistcachestale_test

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"bladeready/internal/web"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskListCacheInvalidatesAfterStateChange(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "task-list-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	httpServer := httptest.NewServer(web.New(application.New(repo)).Handler())
	defer httpServer.Close()

	start := time.Date(2032, time.June, 1, 8, 0, 0, 0, time.UTC)
	created := postBundle(t, httpServer.URL+"/api/tasks", map[string]any{
		"wind_farm":               "缓存复现场",
		"turbine_code":            "CACHE-01",
		"blade_count":             1,
		"inspection_window_start": start,
		"inspection_window_end":   start.Add(2 * time.Hour),
		"created_by":              "测试员",
		"idempotency_key":         "cache-create",
	})

	before := listTasks(t, httpServer.URL+"/api/tasks")
	if len(before) != 1 || before[0].Status != domain.StatusDraft {
		t.Fatalf("failed to prime draft task list: %+v", before)
	}

	updated := postBundle(t, httpServer.URL+"/api/tasks/"+created.Task.ID+"/zones", map[string]any{
		"expected_version": created.Task.Version,
		"actor":            "边界工程师",
		"idempotency_key":  "cache-zones",
		"zones": []map[string]any{{
			"blade_index":          1,
			"zone_code":            "ROOT-A",
			"material_type":        "玻纤复合材料",
			"max_crack_mm":         10,
			"max_delamination_mm":  20,
			"operating_wind_limit": 14,
		}},
	})
	if updated.Task.Status != domain.StatusObserving {
		t.Fatalf("zone write did not advance task: %+v", updated.Task)
	}

	after := listTasks(t, httpServer.URL+"/api/tasks")
	if len(after) != 1 {
		t.Fatalf("unexpected task count after write: %d", len(after))
	}
	if after[0].Status != updated.Task.Status || after[0].Version != updated.Task.Version {
		t.Fatalf("task list cache returned stale snapshot: got status=%s version=%d want status=%s version=%d", after[0].Status, after[0].Version, updated.Task.Status, updated.Task.Version)
	}
}

func postBundle(t *testing.T, url string, payload any) store.TaskBundle {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("POST %s returned %d: %s", url, response.StatusCode, body)
	}
	var bundle store.TaskBundle
	if err = json.Unmarshal(body, &bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func listTasks(t *testing.T, url string) []domain.InspectionTask {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload struct {
		Tasks []domain.InspectionTask `json:"tasks"`
	}
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload.Tasks
}

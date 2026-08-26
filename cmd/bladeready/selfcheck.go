package main

import (
	"bladeready/internal/store"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func executeSelfcheck(base string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	if err := expectGET(client, base+"/api/health", "\"status\":\"ok\""); err != nil {
		return err
	}
	if err := expectGET(client, base+"/", "风叶巡检复机放行台"); err != nil {
		return err
	}
	start := time.Now().UTC().Add(time.Hour)
	end := start.Add(4 * time.Hour)
	b, err := postBundle(client, base+"/api/tasks", map[string]any{"wind_farm": "自检风场", "turbine_code": "SC-01", "blade_count": 3, "inspection_window_start": start, "inspection_window_end": end, "created_by": "自检工程师", "idempotency_key": "selfcheck-create"})
	if err != nil {
		return err
	}
	id := b.Task.ID
	b, err = postBundle(client, base+"/api/tasks/"+id+"/zones", map[string]any{"expected_version": b.Task.Version, "actor": "边界工程师", "idempotency_key": "selfcheck-zones", "zones": []map[string]any{{"blade_index": 1, "zone_code": "ROOT-A", "material_type": "玻纤复合材料", "max_crack_mm": 10, "max_delamination_mm": 25, "operating_wind_limit": 14}, {"blade_index": 2, "zone_code": "ROOT-A", "material_type": "玻纤复合材料", "max_crack_mm": 10, "max_delamination_mm": 25, "operating_wind_limit": 14}, {"blade_index": 3, "zone_code": "ROOT-A", "material_type": "玻纤复合材料", "max_crack_mm": 10, "max_delamination_mm": 25, "operating_wind_limit": 14}}})
	if err != nil {
		return err
	}
	zone := b.Zones[0]
	b, err = postBundle(client, base+"/api/tasks/"+id+"/observations", map[string]any{"expected_version": b.Task.Version, "actor": "无人机巡检员", "idempotency_key": "selfcheck-observation", "observation": map[string]any{"blade_zone_id": zone.ID, "captured_at": time.Now().UTC(), "defect_type": "crack", "position_mm": 1200, "length_mm": 6, "width_mm": 2, "photo_digest": "selfcheck-photo-digest", "review_note": "自检清晰影像", "sequence": 1}})
	if err != nil {
		return err
	}
	obs := b.Observations[0]
	b, err = postBundle(client, base+"/api/tasks/"+id+"/assess", map[string]any{"expected_version": b.Task.Version, "actor": "风险工程师", "idempotency_key": "selfcheck-assess", "planned_wind": 12})
	if err != nil {
		return err
	}
	if b.Assessment == nil || len(b.Assessment.Results) != 1 {
		return fmt.Errorf("自检风险结果缺失")
	}
	b, err = postBundle(client, base+"/api/tasks/"+id+"/repair-plan", map[string]any{"expected_version": b.Task.Version, "actor": "维修负责人", "idempotency_key": "selfcheck-plan", "actions": []map[string]any{{"observation_id": obs.ID, "action": "结构补强", "owner": "维修组", "retest_point": "缺陷中心与周边"}}})
	if err != nil {
		return err
	}
	b, err = postBundle(client, base+"/api/tasks/"+id+"/retests", map[string]any{"expected_version": b.Task.Version, "actor": "复测工程师", "idempotency_key": "selfcheck-retest", "readings": []map[string]any{{"observation_id": obs.ID, "blade_zone_id": zone.ID, "measured_at": time.Now().UTC(), "length_mm": 1, "width_mm": 0.2, "evidence_digest": "selfcheck-retest-evidence", "operator": "复测工程师"}}})
	if err != nil {
		return err
	}
	if len(b.Deviations) != 0 {
		return fmt.Errorf("自检复测意外产生偏差")
	}
	b, err = postBundle(client, base+"/api/tasks/"+id+"/review", map[string]any{"expected_version": b.Task.Version, "actor": "安全复核员", "idempotency_key": "selfcheck-review"})
	if err != nil {
		return err
	}
	b, err = postBundle(client, base+"/api/tasks/"+id+"/release", map[string]any{"expected_version": b.Task.Version, "reviewer": "安全复核员", "idempotency_key": "selfcheck-release", "confirm_boundary": true, "confirm_retests": true, "confirm_audit": true})
	if err != nil {
		return err
	}
	if b.Credential == nil || b.Task.Status != "released" {
		return fmt.Errorf("自检未获得放行凭证")
	}
	resp, err := client.Get(base + "/api/tasks/" + id + "/audit")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("审计接口状态 %d", resp.StatusCode)
	}
	return nil
}

func expectGET(client *http.Client, url, marker string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s 返回 %d", url, resp.StatusCode)
	}
	if !bytes.Contains(body, []byte(marker)) {
		return fmt.Errorf("%s 响应缺少完整性标记 %q", url, marker)
	}
	return nil
}

func postBundle(client *http.Client, url string, payload any) (store.TaskBundle, error) {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return store.TaskBundle{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return store.TaskBundle{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return store.TaskBundle{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return store.TaskBundle{}, fmt.Errorf("%s 返回 %d: %s", url, resp.StatusCode, string(body))
	}
	var b store.TaskBundle
	if err = json.Unmarshal(body, &b); err != nil {
		return b, err
	}
	return b, nil
}

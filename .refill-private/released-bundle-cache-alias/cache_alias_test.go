package released_bundle_cache_alias_test

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"context"
	"encoding/json"
	"testing"
	"time"
)

type snapshotRepository struct {
	template store.TaskBundle
	loads    int
}

func (r *snapshotRepository) LoadTask(context.Context, string) (store.TaskBundle, error) {
	r.loads++
	raw, err := json.Marshal(r.template)
	if err != nil {
		return store.TaskBundle{}, err
	}
	var result store.TaskBundle
	err = json.Unmarshal(raw, &result)
	return result, err
}

func (*snapshotRepository) CreateTask(context.Context, domain.InspectionTask, domain.Event, string, []byte) ([]byte, error) {
	panic("unexpected CreateTask")
}
func (*snapshotRepository) ListTasks(context.Context) ([]domain.InspectionTask, error) {
	panic("unexpected ListTasks")
}
func (*snapshotRepository) SaveBundle(context.Context, store.TaskBundle, domain.Event, string, []byte) ([]byte, error) {
	panic("unexpected SaveBundle")
}
func (*snapshotRepository) Events(context.Context, string) ([]domain.Event, error) {
	panic("unexpected Events")
}
func (*snapshotRepository) Idempotent(context.Context, string) ([]byte, bool, error) {
	panic("unexpected Idempotent")
}
func (*snapshotRepository) Check(context.Context) error { return nil }
func (*snapshotRepository) Close() error                { return nil }

func TestReleasedBundleCacheDoesNotLeakCallerMutations(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := &snapshotRepository{template: store.TaskBundle{
		Task:  domain.InspectionTask{ID: "task-released", Status: domain.StatusReleased},
		Zones: []domain.BladeZone{{ID: "zone-1", ZoneCode: "ROOT"}},
		Assessment: &domain.AssessmentSnapshot{
			TaskID: "task-released",
			Results: []domain.RiskResult{{
				ObservationID:         "obs-1",
				Reasons:               []string{"原始风险原因"},
				SuggestedRetestPoints: []string{"ROOT@100mm"},
				RuleHits:              []domain.RuleHit{{Rule: "crack", Explanation: "原始规则解释"}},
			}},
			ZoneSummaries: []domain.RiskZoneSummary{{BladeZoneID: "zone-1", TotalScore: 30}},
		},
		Credential:   &domain.ReleaseCredential{ID: "credential-1", Reviewer: "安全复核员", IssuedAt: now},
		ZoneCoverage: map[int]int{1: 1},
	}}
	svc := application.New(repo)

	first, err := svc.Get(context.Background(), "task-released")
	if err != nil {
		t.Fatal(err)
	}
	first.Zones[0].ZoneCode = "TAMPERED"
	first.Assessment.Results[0].Reasons[0] = "被调用方改写"
	first.Assessment.Results[0].RuleHits[0].Explanation = "被调用方改写"
	first.Assessment.ZoneSummaries[0].TotalScore = 999
	first.Credential.Reviewer = "被调用方改写"
	first.ZoneCoverage[1] = 999

	second, err := svc.Get(context.Background(), "task-released")
	if err != nil {
		t.Fatal(err)
	}
	if repo.loads != 1 {
		t.Fatalf("预期命中已放行缓存，实际加载存储 %d 次", repo.loads)
	}
	if second.Zones[0].ZoneCode != "ROOT" ||
		second.Assessment.Results[0].Reasons[0] != "原始风险原因" ||
		second.Assessment.Results[0].RuleHits[0].Explanation != "原始规则解释" ||
		second.Assessment.ZoneSummaries[0].TotalScore != 30 ||
		second.Credential.Reviewer != "安全复核员" || second.ZoneCoverage[1] != 1 {
		t.Fatalf("缓存聚合被前一个调用方污染: %+v", second)
	}
}

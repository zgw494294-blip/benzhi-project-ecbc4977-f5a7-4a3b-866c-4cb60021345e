package canceledwritecontext_test

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"bladeready/internal/web"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestCanceledWriteRequestDoesNotPersistState(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "canceled-write.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	service := application.New(repo)
	now := time.Now().UTC()
	bundle, err := service.CreateTask(context.Background(), application.CreateTaskRequest{
		WindFarm:              "取消传播测试风场",
		TurbineCode:           "CTX-01",
		BladeCount:            1,
		InspectionWindowStart: now,
		InspectionWindowEnd:   now.Add(time.Hour),
		CreatedBy:             "测试员",
		IdempotencyKey:        "ctx-create",
	})
	if err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(application.SetZonesRequest{
		ExpectedVersion: bundle.Task.Version,
		Actor:           "测试员",
		IdempotencyKey:  "ctx-zones",
		Zones: []domain.BladeZone{{
			BladeIndex:         1,
			ZoneCode:           "ROOT",
			MaterialType:       "玻纤",
			MaxCrackMM:         10,
			MaxDelaminationMM:  20,
			OperatingWindLimit: 14,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+bundle.Task.ID+"/zones", bytes.NewReader(payload)).WithContext(requestContext)
	recorder := httptest.NewRecorder()
	web.New(service).Handler().ServeHTTP(recorder, request)

	stored, err := repo.LoadTask(context.Background(), bundle.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Task.Status != domain.StatusDraft || stored.Task.Version != bundle.Task.Version || len(stored.Zones) != 0 {
		t.Fatalf("canceled request persisted state: status=%s version=%d zones=%d response_status=%d", stored.Task.Status, stored.Task.Version, len(stored.Zones), recorder.Code)
	}
}

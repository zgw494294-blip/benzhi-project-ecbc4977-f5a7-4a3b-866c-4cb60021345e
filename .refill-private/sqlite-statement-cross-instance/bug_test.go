package sqlite_statement_cross_instance_test

import (
	"bladeready/internal/application"
	"bladeready/internal/store"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLitePreparedStatementBelongsToRepositoryInstance(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.August, 26, 8, 0, 0, 0, time.UTC)

	firstRepo, err := store.Open(filepath.Join(t.TempDir(), "first.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer firstRepo.Close()
	firstService := application.New(firstRepo)
	first, err := firstService.CreateTask(ctx, application.CreateTaskRequest{
		WindFarm:              "甲风场",
		TurbineCode:           "A-01",
		BladeCount:            1,
		InspectionWindowStart: now,
		InspectionWindowEnd:   now.Add(time.Hour),
		CreatedBy:             "甲",
		IdempotencyKey:        "create-first",
	})
	if err != nil {
		t.Fatal(err)
	}

	secondRepo, err := store.Open(filepath.Join(t.TempDir(), "second.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer secondRepo.Close()
	secondService := application.New(secondRepo)
	if _, err = secondService.CreateTask(ctx, application.CreateTaskRequest{
		WindFarm:              "乙风场",
		TurbineCode:           "B-01",
		BladeCount:            1,
		InspectionWindowStart: now.Add(2 * time.Hour),
		InspectionWindowEnd:   now.Add(3 * time.Hour),
		CreatedBy:             "乙",
		IdempotencyKey:        "create-second",
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := firstService.Get(ctx, first.Task.ID)
	if err != nil {
		t.Errorf("打开第二个仓库后，第一个仓库无法读取自己的任务: %v", err)
	} else if loaded.Task.ID != first.Task.ID || loaded.Task.WindFarm != "甲风场" {
		t.Errorf("第一个仓库读取到其他实例的数据: %+v", loaded.Task)
	}

	if err = secondRepo.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = firstService.Get(ctx, first.Task.ID); err != nil {
		t.Errorf("关闭第二个仓库后，第一个仓库的查询资源失效: %v", err)
	}
}

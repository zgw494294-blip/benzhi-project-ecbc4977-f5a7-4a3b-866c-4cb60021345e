# 风叶巡检复机放行台

本项目是面向风电场巡检工程师、叶片维修负责人和安全复核人员的浏览器工作台。它把巡检建案、叶片运行边界冻结、无人机缺陷观测、规则风险分级、维修计划、维修后复测、偏差整改和复机放行凭证串成一条可追溯流程。

服务使用 Go 原生 HTTP 页面和同源 JSON API，数据保存在本地 SQLite。数据库启用 WAL、外键和完整性检查；每次写操作使用 `expected_version` 做乐观并发校验，使用 `idempotency_key` 避免重复提交，放行凭证签发后由数据库触发器保持不可变。

## 构建与运行

```text
go build ./cmd/bladeready
go run ./cmd/bladeready -addr=127.0.0.1:19081 -db=bladeready.db
```

浏览器打开 `http://127.0.0.1:19081/`。监听地址也可通过 `PORT` 环境变量设置；例如 `PORT=19100` 会监听 `127.0.0.1:19100`。显式 `-addr` 优先于 `PORT`。默认只监听高位回环地址 `127.0.0.1:19081`。

## 测试与自检

```text
go test ./...
go run ./cmd/bladeready -selfcheck -addr=127.0.0.1:19081
```

`-selfcheck` 使用临时 SQLite 数据库并实际启动 HTTP 监听，然后经 JSON API 自动完成从创建任务到签发凭证的全链路，结束后删除临时数据并自行退出。

## 主要 API

任务由 `POST /api/tasks` 创建，通过 `/api/tasks/{id}/zones`、`observations`、`assess`、`repair-plan`、`retests`、`deviations`、`review` 和 `release` 依次推进。`GET /api/tasks/{id}/audit` 返回有序审计事件，`GET /api/tasks/{id}/credential` 返回最终不可变凭证。所有写入接口均返回最新任务聚合及版本。

`GET /api/tasks` 支持 `status`、`wind_farm`、`from`、`to`（RFC3339）筛选，并为草稿或观测窗口已结束的任务返回 `overdue` 标记。`GET /api/tasks/{id}` 可使用 `level`（或 `risk_level`）和 `blocked` 查询风险命中结果；任务详情及审计事件包含冻结边界摘要、逐片覆盖索引和分区风险统计。

# BENZHI_README

基于 Go 实现的风电叶片巡检复机放行 Web 项目，一款后端服务，用于管理风电叶片巡检、缺陷评估、维修复测和复机放行。

## 项目说明
- 项目：benzhi-project-ecbc4977-f5a7-4a3b-866c-4cb60021345e
- 项目用途：用于支持风叶巡检复机放行台的核心业务流程。
- Go 工具链：`golang:1.23`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/bladeready -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-ecbc4977-f5a7-4a3b-866c-4cb60021345e-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-ecbc4977-f5a7-4a3b-866c-4cb60021345e-arm64 linux/arm64
docker run -it benzhi-project-ecbc4977-f5a7-4a3b-866c-4cb60021345e-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/bladeready -selfcheck -addr=127.0.0.1:19081`

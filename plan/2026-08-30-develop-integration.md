# develop 分支完整集成计划

## 目标

以最新 `origin/main` 为基线，将 Platform API 合同传播、本地 WeKnora 构建和请求上下文日志能力迁移到 `develop`。

## 实施步骤

1. 迁移 `feat/dixian-agent-integration-20260829` 的唯一功能提交。
2. 保留生成合同的单一来源，检查中间件修改与现有行为兼容。
3. 执行 Go 格式化、单元测试和 Docker 构建验证。

## 影响范围

- Platform API v1 合同及生成 Go 类型。
- 请求日志中间件。
- 本地 WeKnora Docker 镜像。

## 验证方法

- `go test ./...`
- `go test ./internal/middleware ./internal/dixiancontract`
- `docker build -f docker/Dockerfile.dixian-local .`

## 进度

- [ ] 功能迁移
- [ ] 自动化验证
- [ ] 推送远程 develop

## 最终结果

待实施完成后更新。

# T04 Agent 契约同步计划

## 目标

让 Knowledge 仓库内保存的 Platform API 契约与 T04 Agent 版本、测试、发布接口保持一致。本仓库不实现 Agent 业务，只保留自动生成的 Go 请求与响应类型。

## 实施步骤

- [x] 从 Platform OpenAPI 同步 T04 Agent API 契约。
- [x] 重新生成 Go 契约类型。
- [x] 运行契约一致性检查和聚焦 Go 测试。
- [x] 记录最终结果。

## 影响范围

- `contracts/dixian/platform-api-v1.json`
- `internal/dixiancontract/platform_api_v1.go`

## 验证方法

- Platform 契约同步检查无差异。
- Go 生成文件可格式化并通过聚焦测试。

## 进度

已完成。

## 最终结果

Platform OpenAPI、Knowledge 仓库契约副本和 Go 生成类型已同步。Platform 契约检查与 `go test ./internal/dixiancontract ./internal/middleware` 通过。

# develop 分支安全集成计划

## 目标

以最新 `origin/main` 为基线，将 Platform API 合同、请求上下文日志和本地 WeKnora 部署能力完整纳入 `develop`，同时保留无 Ollama 的本地部署要求。

## 实施步骤

1. 抓取并核对全部远程分支、标签和提交关系。
2. 保存未完成工作的本地安全快照。
3. 保留功能提交，迁移已经确认的无 Ollama 本地配置。
4. 执行 Go 格式化、相关单元测试和 Docker 配置验证。
5. 通过代码检查后再提交并创建远程 `develop`。

## 影响范围

- Platform API v1 合同及生成 Go 类型。
- 请求日志中间件。
- 本地 WeKnora Docker Compose 与启动说明。

## 验证方法

- `go test ./internal/middleware ./internal/dixiancontract`
- `docker compose config`
- 本地容器健康检查和 API 链路验证。

## 进度

- [x] 全部远程引用已抓取并核对
- [x] 未完成工作已保存到本地安全分支
- [x] 功能提交已迁移且与远程补丁等价
- [ ] 无 Ollama 配置迁移
- [ ] 自动化验证
- [ ] 代码检查
- [ ] 创建远程 develop

## 最终结果

待验证和远程更新完成后填写。

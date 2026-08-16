# dx-docs (WeKnora) 开发约定

本文件是 WeKnora RAG 知识库平台项目的开发指南。优先响应当前请求；未提供特殊说明时遵循以下约定。

## 项目概览

WeKnora 是 RAG 知识库平台：

- **后端**: Go, Gin/Echo, PostgreSQL, Redis, MinIO, Neo4j, gRPC
- **前端**: React, TypeScript, Vite
- **文档读取**: Python 微服务 (docreader/), gRPC 通信
- **部署**: Docker Compose, Helm Chart

主要目录：

| 目录 | 用途 |
|------|------|
| `internal/` | Go 后端核心业务逻辑 |
| `cmd/` | 应用入口 |
| `config/` | 配置定义 |
| `frontend/` | 前端代码 |
| `docreader/` | 文档读取微服务 |
| `migrations/` | 数据库迁移 |
| `deploy/` | 部署配置 |
| `docs/` | 项目文档 |
| `tests/` | 测试 |

## 常用命令

```bash
# 快速开发模式（推荐）
make dev-start        # 启动基础设施 (PostgreSQL, Redis, MinIO, Neo4j, DocReader)
make dev-app          # 启动 Go 后端 (支持 Air 热重载)
make dev-frontend     # 启动前端 (Vite 热重载)
make dev-status       # 查看服务状态
make dev-stop         # 停止所有服务

# 构建
sh scripts/build_images.sh          # 构建所有镜像
sh scripts/build_images.sh -p       # 仅后端
sh scripts/build_images.sh -f       # 仅前端
sh scripts/build_images.sh -d       # 仅文档读取器

# 测试
go test ./...                         # 运行所有 Go 测试
cd frontend && npm run test           # 前端测试
```

## 开发规范

- 编辑前先阅读现有模块，保持当前架构、命名和代码风格。
- Go 代码遵循 `go.mod` 中的依赖管理和 `.golangci.yml` 中的 lint 规则。
- 前端使用 npm/pnpm，不提交 `node_modules/` 或构建产物。
- 后端使用 Air 热重载（项目已内置 `.air.toml`），修改后自动重启。
- 前端使用 Vite 热重载，修改后自动刷新。
- 对 auth、RBAC、知识库权限、文件上传、Agent Skills 执行等敏感路径，采用保守变更并添加验证。
- 新配置必须加入 schema 校验；关键密钥必填且无默认值。
- 不打印或提交令牌、密钥、凭据或连接串。
- 数据库 schema 变更使用 migrations/ 目录中的迁移文件。

## 本地开发地址

- 前端: http://localhost:5173
- 后端 API: http://localhost:8080
- MinIO Console: http://localhost:9001
- Neo4j Browser: http://localhost:7474

## 验证指南

| 变更类型 | 验证方式 |
|----------|----------|
| Go 逻辑 | `go test ./...` (相关测试) |
| Go 格式 | `golangci-lint run` |
| 前端逻辑 | `cd frontend && npm run test` |
| 前端构建 | `cd frontend && npm run build` |
| API 契约 | Swagger UI 验证 (`docs/swagger.yaml`) |
| 数据库 | 运行迁移，确认 schema 正确 |
| 文档 | 确认 Markdown 链接、命令和路径正确 |

验证因缺少服务或环境变量无法完成时，明确说明。

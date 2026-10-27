# dx-docs 开发约定

dx-docs 是仅供 Platform 调用的帝显知识服务。**Dixian 业务智能体由 dixian-app 的 `search` graph 执行，本仓库不再执行 native 业务智能体。**

## 三仓库职责与交接（2026-10-27）

- **dixian-platform**：用户入口、权限、智能体版本及知识工具调用授权，是身份与资源 ACL 的唯一权威。
- **dixian-app**：`search` graph 执行业务智能体，负责模型调用、聊天附件、沙箱及文件交付。
- **dx-docs**：负责知识库、文档解析/分块、检索和 graph/wiki 等知识工具；保留受控 tool bridge，每次调用按授权知识库及文档归属校验作用域。
- native execute 路由已移除；`native_config`、`/agent-config`、`/agents/:id` 以及遗留引擎代码是历史兼容档案/能力，不是业务执行开关。不得通过恢复路由或修改 workflow 重新启用 native。

跨仓库架构权威说明见 [Platform 架构文档](https://github.com/AipengHuang/dixian-platform/blob/main/docs/architecture/README.md)。本仓库现状、路由证据、部署与验收边界见 [DEVELOPMENT.md](./DEVELOPMENT.md)；开发入口：[Platform](https://github.com/AipengHuang/dixian-platform/blob/main/AGENTS.md)、[Runtime/App](https://github.com/AipengHuang/dixian-app/blob/main/AGENTS.md)。

上游 `../AGENTS.md` 的项目记忆和本仓库 `plan/` 记录了 native 时代故障与修复；其执行链路、切换方法和旧部署说明属于历史背景。与本节冲突时，以当前代码及上述交接文档为准，不照搬旧 SQL 切换或部署手法。

## 目录

| 目录 | 用途 |
| --- | --- |
| `internal/` | Go 后端核心业务逻辑 |
| `cmd/` | 服务入口 |
| `config/` | 配置定义 |
| `docreader/` | 文档读取服务 |
| `migrations/` | 数据库迁移 |
| `helm/` | 私网部署配置 |

## 命令

```bash
make dev-start
make dev-app
go test ./...
golangci-lint run
```

## 规则

- 除 `/health` 外，只提供 `/internal/v1` 私有知识与兼容配置接口，不提供浏览器前端、用户登录、用户 Token 或公开 API。
- 知识工具复用现有 WeKnora 实现，必须保留 Platform 的调用者与授权上下文、每次调用的作用域校验；不得使用创建者权限或把兼容配置当作执行授权。
- Platform 是用户、组织、角色、权限和资源 ACL 的唯一权威。
- 修改认证、知识权限、上传和检索边界时必须添加拒绝越权的测试。
- 新配置必须校验；关键密钥必填且无默认值。
- 不打印或提交令牌、密钥、凭据或连接串。
- 数据库结构变化使用 `migrations/`。
- Go 代码使用 `gofmt`，验证以 `go test ./...` 为准。

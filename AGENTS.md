# dx-docs 开发约定

dx-docs 是仅供 Platform 调用的帝显知识与原生智能体执行服务。

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

- 只提供 `/internal/v1` 知识与智能体接口，不提供浏览器前端、用户登录、用户 Token 或公开 API。
- 智能体直接复用现有配置和执行引擎；原始调用者、冻结版本和运行权限来自 Platform，每次检索、模型调用和工具操作须重验，不得使用创建者权限。
- Platform 是用户、组织、角色、权限和资源 ACL 的唯一权威。
- 修改认证、知识权限、上传和检索边界时必须添加拒绝越权的测试。
- 新配置必须校验；关键密钥必填且无默认值。
- 不打印或提交令牌、密钥、凭据或连接串。
- 数据库结构变化使用 `migrations/`。
- Go 代码使用 `gofmt`，验证以 `go test ./...` 为准。

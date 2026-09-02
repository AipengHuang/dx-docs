# T03 Knowledge 授权上下文实施计划

## 目标

Knowledge 服务验证 Platform 签名的版本 1 授权上下文，并以其中的真实用户作为调用主体。内部请求不再把该用户提升为 Owner。

## 最小实现

- 使用 Go 标准库验证 HMAC、Base64URL、JSON、版本、有效期和服务绑定，不增加依赖。
- 继续精确绑定操作、组织、资源和 Log Number。
- Platform 已授权的内部主体使用最低 Viewer 角色；具体内部路由只由一次性 operation grant 放行，不继承 Owner 能力。
- 不改公开知识页面或新增配置项。

## 实施步骤

- [x] 写失败测试，覆盖签名篡改、错误版本、过期、跨服务、错误操作和错误资源。
- [x] 解析并验证 Platform 授权 token。
- [x] 把真实 assignment、权限和 Scope 保存到请求上下文，供检索与审计使用。
- [x] 删除 Platform 调用用户的 Owner 提权。
- [x] 运行 gofmt、聚焦测试、全量 Go 测试和 `ai-code-check`。

## 影响范围

- `internal/middleware/platform_internal.go`
- `internal/middleware/platform_internal_test.go`

## 验证方法

- 非 Platform 服务、错误签名、错误版本、过期或字段不匹配全部拒绝。
- 通过后的知识请求主体仍是真实 Platform 用户。
- 下游上下文能读取本次实际 assignment、权限码和 Scope。

## 进度

已完成。细粒度列表过滤在知识元数据映射完成前不启用，Platform 对这类 Scope 已统一 fail closed。安全复核未发现新的重要问题。

## 最终结果

已验证 Platform 签名、版本、时间、audience、操作、组织、资源和 Log Number，并以真实 Platform 用户和 Viewer 角色执行。gofmt、差异检查和全量 `go test ./...` 通过；仅有重复 `-lc++` 的现有链接警告。

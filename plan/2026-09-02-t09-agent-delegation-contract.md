# T09 跨 Agent 契约同步计划

## 目标

同步 Platform 的 Agent 发布契约和跨 Agent 委派边界，确保 Runtime 与知识服务不会自行推断权限或用户身份。

## 实施步骤

- [x] 同步 Platform OpenAPI 契约。
- [x] 补充委派链路说明和边界说明。
- [x] 校验浏览器仍只调用 Platform 公共 API，Runtime 仅使用内部服务接口。

## 影响范围

- `contracts/dixian/platform-api-v1.json`
- `README.md`

## 验证方式

- 契约文件逐字一致性检查。
- 文档边界关键字检查。

## 进度

- [x] 完成现状分析。
- [x] 同步完成。
- [x] 验证完成。

## 最终结果

Platform OpenAPI 镜像已同步，README 已明确跨 Agent 授权不下沉到知识服务。契约一致性检查已通过。

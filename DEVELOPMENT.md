# Dixian Knowledge Development

## Current Dixian execution boundary

The authoritative cross-repository handoff is the
[Platform architecture](https://github.com/AipengHuang/dixian-platform/blob/main/docs/architecture/README.md).

As of 2026-10-27, Platform owns the user entry point, identities, RBAC/ACL,
Agent versions, and authorization of each knowledge-tool call. **dixian-app's
`search` graph runs business Agents**, including model calls, chat attachments,
sandbox work, and generated-file delivery. dx-docs supplies knowledge storage,
parsing, chunks, retrieval, and document/graph/wiki tools through Platform.

The business execution path is Platform → dixian-app `search`; the knowledge
tool path is Runtime → Platform authorization → dx-docs → scoped knowledge data.
Knowledge-service parsing/indexing capabilities remain here; this is not a
replacement business Agent execution loop.

### Native compatibility is not execution

- `/internal/v1/agents/:id/execute` and native tool-approval execution routes are
  no longer registered. See [router.go](./internal/router/router.go) and
  [the retirement regression test](./internal/router/knowledge_tools_test.go).
- `/internal/v1/agent-config`, its validation endpoint, and `/internal/v1/agents/:id`
  configuration routes remain for compatibility. `native_config` and retained
  WeKnora engine/model configuration are historical compatibility records, not
  a switch to run business Agents. Do not restore native routes or workflow.
- The active bridge is
  `POST /internal/v1/knowledge-bases/:id/tools/:tool`. Each request redeems the
  Platform grant and verifies its operation, resource and target. The handler
  limits tool/knowledge actions to that grant and checks document membership in
  the granted knowledge base; arguments cannot expand it. See
  [route registration](./internal/router/internal_knowledge_routes.go) and
  [KnowledgeToolsHandler](./internal/handler/knowledge_tools.go).
- Existing `plan/`, built-in Agent/model configuration, and generic WeKnora
  engine documentation remain useful historical or implementation references.
  Their native execution and attachment flow descriptions do not apply to
  current Dixian business runs. Do not delete them merely to document retirement.

### Deployment and acceptance snapshot — 2026-10-27

- Deployed migration: dx-docs [PR #15](https://github.com/AipengHuang/dx-docs/pull/15)
  (`2744754c`), Platform [PR #51](https://github.com/AipengHuang/dixian-platform/pull/51),
  and App [PR #73](https://github.com/AipengHuang/dixian-app/pull/73) plus
  [PR #75](https://github.com/AipengHuang/dixian-app/pull/75) (`f9367716`).
  The migration covered 23 Agent versions and 46 sessions.
- The 9/9 result is a **smoke check only**, not full business acceptance.
  The user restored model quota and explicitly resumed business testing on
  2026-10-27. Earlier quota-pause notes are historical, not a current instruction.
- App [PR #77](https://github.com/AipengHuang/dixian-app/pull/77) was deployed as
  `f92e7bea`; its event-flush and historical-error fixes have been checked.
  Follow-up answer, workbook-preservation and citation fixes are tracked in
  App [PR #79](https://github.com/AipengHuang/dixian-app/pull/79).
- Current business outcomes and remaining verification boundaries live in the
  [central cross-repository handoff](https://github.com/AipengHuang/dixian-platform/blob/main/docs/architecture/README.md#2026-10-27-交接快照与验证边界).
  Do not duplicate changing acceptance counts here or equate merged code, green
  CI, a healthy deployment and verified model answers.
- All three repositories now deploy their own services automatically from
  `main`, sharing `/opt/dixian/.deploy.lock` on the deployment host. This repo's
  [workflow](./.github/workflows/deploy.yml) builds knowledge/docreader images,
  then pulls and recreates only those services under the lock and checks health.
  Do not use the historical workaround of pushing an unrelated Platform commit
  to deploy App, or assume a `main` documentation merge has no deployment effect.

## Entry point

`cmd/server` starts a private HTTP service. `internal/router/router.go`
registers only `/health` and `/internal/v1`.

## Authorization flow

1. Platform authenticates the browser session and checks RBAC plus knowledge ACL.
2. Platform issues a one-time execution handle bound to one organization,
   operation, resource, and Log Number.
3. This service validates the Platform service credential and redeems the
   handle through Platform.
4. The request runs in the organization workspace. Any mismatch returns 403.

There is no fallback login, JWT, API key, invitation, tenant role, or
RBAC-disabled bypass on the registered HTTP surface.

## Commands

```bash
gofmt -w <changed-go-files>
go test ./internal/middleware ./internal/router
make test
make build
```

The unified React frontend and deployment compose live in the App and Platform
repositories respectively.

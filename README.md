# Dixian Knowledge Service

This repository is the private knowledge execution service for the Dixian AI
platform. It has no public login, user center, role system, browser frontend,
or human API token.

The browser uses the Platform `/api/v1` entry point. Business Agents run in
the **dixian-app `search` graph**: Runtime handles model calls, chat attachments,
sandbox execution, and generated-file delivery. Platform owns identities,
permissions, Agent versions, and knowledge-tool authorization. Platform authorizes
each knowledge operation, creates a one-time execution handle, and calls this
service through `/internal/v1`.

**Dixian no longer runs business Agents in the WeKnora native engine.** Native
execute routes are removed. Retained `native_config`, Agent configuration routes,
and engine source are compatibility/history, not switches to restore native runs.

## Service boundary

- Platform is the authority for users, organizations, roles, permissions,
  knowledge ACLs, navigation, and audit correlation.
- This service owns knowledge storage, parsing, chunks, FAQ, Wiki, retrieval,
  citations, and data-source execution.
- The knowledge tool bridge retains document/chunk/graph/wiki tools. Each call
  is restricted to the granted knowledge base and its documents; it does not
  start a native Agent loop.
- Every request requires the Platform service identity, a short-lived
  one-time handle, organization ID, resource ID, request ID, and Log Number.
- Missing or mismatched context is rejected.
- Cross-Agent discovery, approval, temporary RBAC access, execution, and audit
  remain in Platform and Runtime. This service receives only the same real user,
  authorized scopes, and Log Number when a delegated Agent retrieves knowledge.

For the authoritative cross-repository architecture, see the
[Platform architecture](https://github.com/AipengHuang/dixian-platform/blob/main/docs/architecture/README.md).
For the current deployment and acceptance snapshot, see
[DEVELOPMENT.md](./DEVELOPMENT.md). AI contributors should start with
[AGENTS.md](./AGENTS.md), then the
[Platform instructions](https://github.com/AipengHuang/dixian-platform/blob/main/AGENTS.md)
and [Runtime instructions](https://github.com/AipengHuang/dixian-app/blob/main/AGENTS.md).
Older plans and inherited WeKnora engine documentation describe historical or
generic capabilities; they do not define the current Dixian execution path.

## Local verification

```bash
make test
make build
```

Use `dixian-platform/compose.local.yml` to run the complete local stack.

## License

The original MIT license and copyright notices remain in [LICENSE](./LICENSE).

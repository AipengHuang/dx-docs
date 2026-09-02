# Dixian Knowledge Service

This repository is the private knowledge execution service for the Dixian AI
platform. It has no public login, user center, role system, browser frontend,
or human API token.

The browser calls only the Platform `/api/v1` API. Platform authorizes each
operation, creates a one-time execution handle, and calls this service through
`/internal/v1`.

## Service boundary

- Platform is the authority for users, organizations, roles, permissions,
  knowledge ACLs, navigation, and audit correlation.
- This service owns knowledge storage, parsing, chunks, FAQ, Wiki, retrieval,
  citations, and data-source execution.
- Every request requires the Platform service identity, a short-lived
  one-time handle, organization ID, resource ID, request ID, and Log Number.
- Missing or mismatched context is rejected.
- Cross-Agent discovery, approval, temporary RBAC access, execution, and audit
  remain in Platform and Runtime. This service receives only the same real user,
  authorized scopes, and Log Number when a delegated Agent retrieves knowledge.

## Local verification

```bash
make test
make build
```

Use `dixian-platform/compose.local.yml` to run the complete local stack.

## License

The original MIT license and copyright notices remain in [LICENSE](./LICENSE).

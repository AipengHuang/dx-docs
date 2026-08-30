# Dixian Knowledge Development

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

# T22 Registered knowledge connectors

## Objective

Return only connectors that are actually registered in the running dx-docs container so the product never advertises unavailable integrations.

## Implementation steps

- [ ] Expose registered connector types from the existing registry through the data-source service.
- [ ] Resolve metadata only for those exact registered types.
- [ ] Keep output deterministic and add a focused regression test.
- [ ] Run Go formatting, focused tests, and `ai-code-check`.

## Affected areas

- `internal/datasource/connector.go`
- `internal/application/service/datasource_service.go`
- `internal/types/interfaces/datasource.go`
- `internal/handler/datasource.go`

## Verification

- Verify an unregistered metadata entry never appears.
- Verify all registered connector entries keep their typed metadata.

## Progress

In progress.

## Final outcome

Pending implementation and verification.

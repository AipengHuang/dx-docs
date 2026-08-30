# Dixian Document Reader

This private gRPC service parses documents for the Dixian Knowledge Service.
It is not exposed to browsers and has no authentication or user interface.

## Runtime

- Port: `50051`
- Health check: `grpc_health_probe -addr=127.0.0.1:50051`
- Deployment: `dixian-platform/compose.local.yml`

## Verification

```bash
cd docreader
uv run pytest
```

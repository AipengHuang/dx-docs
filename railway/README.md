# Dixian Railway deployment

The `dixian` Railway project is isolated from all Adax projects. It contains:

- `dixian-app`: branded API/agent runtime built from this repository.
- `dixian-ui`: branded frontend built from `frontend/`.
- `dixian-docreader`: private gRPC document parser.
- `dixian-paradedb`: private ParadeDB with a persistent volume.
- `dixian-redis`: private Redis with a persistent volume.

The App also mounts a persistent volume at `/data/files`. Only App and UI have
public Railway domains. Database, Redis, and DocReader remain private.

Runtime credentials are Railway service variables. The tracked model file only
contains `${DEEPSEEK_API_KEY}` and `${JINA_API_KEY}` references; secret values
must never be committed.

GitHub Actions runs App, frontend, and DocReader verification. The three
source-backed Railway services have `source.checkSuites=true`, so Railway waits
for the complete `CI/CD` check suite before deploying a commit from `main`.

Railway configuration files:

- App: `/railway/app.toml`
- DocReader: `/railway/docreader.toml`
- UI: `/frontend/railway.toml`

Production defaults intentionally disable the Docker sandbox because Railway
does not expose a host Docker daemon to application containers.

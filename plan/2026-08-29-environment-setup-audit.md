# Environment Setup and Audit

## Objective

Install the locked Go, frontend, DocReader, and MCP server dependencies needed for local development, identify required environment variables without exposing secret values, configure the requested local-only Docker environment, and make the health endpoint satisfy the Dixian platform tracing contract.

## Implementation Steps

- Inspect repository guidance, manifests, lockfiles, and environment examples.
- Install the Go toolchain required by `go.mod`, then download module dependencies.
- Install frontend dependencies from `package-lock.json`.
- Sync DocReader and MCP server Python environments from their uv lockfiles.
- Compare declared configuration with local environment files and runtime usage.
- Create the ignored local `.env` using the user-provided Docker deployment settings.
- Validate Docker Compose interpolation without exposing secret values.
- Install and register Docker Buildx for the local frontend image build.
- Pull or build the default Compose images and start the core services.
- Remove the local Ollama environment entries, Compose service, and persistent volume declaration.
- Replace the legacy startup script with a core Compose manager that has no local-model installation or compatibility path.
- Remove the obsolete Make target and deployment documentation for the local model service.
- Run `ai-code-check` against the deployment configuration change.
- Verify the local Web UI and platform-facing health contract; model conversation is out of scope until a model is configured.
- Validate and echo the platform `X-Log-Number` header through the shared request tracing middleware.
- Run lightweight dependency and configuration checks.

## Affected Areas

- Local Go module cache, `frontend/node_modules/`, and Python `.venv/` directories.
- Local ignored `.env` file.
- Homebrew Docker Buildx installation, Docker CLI plugin configuration, images, containers, networks, and named volumes.
- Local ignored `.env` and committed `docker-compose.yml` Ollama deployment entries.
- `internal/middleware/logger.go` request tracing and its focused tests.
- `scripts/start_all.sh` local runtime lifecycle entrypoint.
- `Makefile` and `DEVELOPMENT.md` local deployment instructions.
- This plan file; no application source or committed secret file is changed.

## Verification

- Confirm locked installs complete without lockfile changes.
- Confirm Go, frontend, DocReader, and MCP server dependencies resolve.
- Confirm `.env` is ignored, contains the expected keys, and produces a valid Compose model.
- Confirm all default containers reach their expected running or healthy state.
- Confirm the loopback Web UI and API health endpoint respond.
- Confirm the Compose model no longer contains an Ollama service or volume and the remote-model core remains healthy.
- Confirm the Web UI loads; do not treat an expected model authorization failure as a broken business dependency.
- Confirm the rebuilt App image echoes a valid platform log number and makes the platform readiness probe pass.
- Report environment variable names and missing status only.

## Progress

- [x] Repository structure and install commands inspected.
- [x] Dependencies installed.
- [x] Environment variables audited.
- [x] Verification completed.
- [x] Local Docker `.env` configured.
- [x] Compose environment validation completed.
- [x] Docker Buildx installed and registered.
- [x] Core Docker services started.
- [x] Runtime health checks completed.
- [x] Local Ollama deployment configuration removed.
- [x] Remote-model-only runtime revalidated.
- [x] Deployment configuration reviewed with `ai-code-check`.
- [x] Local model startup and compatibility entries removed.
- [x] Platform tracing contract verified against the rebuilt local image.
- [x] Web UI and platform login verified.
- [ ] Model conversation deferred until a model is configured.

## Final Outcome

Go `1.26.7`, root and nested module dependencies, frontend npm dependencies, and the DocReader and MCP Python environments are installed. Docker Buildx is installed and the five core services are running. Local Ollama deployment and compatibility paths are removed. The rebuilt App echoes the platform log number and the platform readiness probe passes. Model conversation remains intentionally unverified until a model is configured.

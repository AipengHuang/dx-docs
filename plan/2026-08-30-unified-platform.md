# Internal Knowledge Service

## Objective

Turn `dx-docs` into a private knowledge execution service called only by Platform, with no public human authentication, role authority, Vue product shell, or external WeKnora branding.

## Implementation Steps

- Add one fail-closed Platform service identity middleware and internal API group.
- Reuse the existing knowledge handlers behind the internal boundary.
- Remove public human authentication, tenant-role authority, and duplicate product routes.
- Stop building and publishing the Vue frontend after React feature parity.
- Remove all externally visible WeKnora branding while preserving required licenses.

## Affected Areas

- Go routing, authentication middleware, configuration, and knowledge handlers.
- Frontend and deployment definitions.
- Security, route, and branding tests.

## Verification

- Run focused red-green tests for service identity and organization isolation.
- Run Go tests, lint, and build.
- Verify public auth routes are absent and external artifacts contain no WeKnora branding.

## Progress

- [x] Existing routes, role middleware, knowledge handlers, and branding surface inspected.
- [x] Internal Platform boundary implemented.
- [x] Public human authentication and role routes removed from the shipped server.
- [x] Vue product shell retired.
- [x] External product branding removed from compiled Go code and browser assets.
- [x] Platform organization IDs explicitly map to Docs tenants.
- [x] Runtime container trimmed to knowledge, parsing, search, data-source, storage, and audit dependencies.
- [x] Full Go test suite, full build, and diff checks completed.
- [x] Full Compose images built and all nine services reached healthy status.
- [x] Verify document ingestion with the default embedding and summary models through Platform.
- [x] Verify fresh upload, parsing, summary generation, vector indexing, exact search, download, ACL denial, and public brand removal through the single Web entry point.
- [x] Run `go test ./...` and `go build ./...` to completion in the official Go 1.26 environment; both exited successfully.

## Live Verification Findings

- The first full Compose build exposed two stale Docker assumptions: the DuckDB extension downloader had been deleted and DocReader still copied a removed offline package directory. Both image defects are fixed without restoring legacy product code.
- Real upload reached DocReader and chunking, then stopped because the knowledge base had vector indexing enabled with an empty `embedding_model_id` even though one active default Embedding model already exists.
- Keep Docs model selection explicit: Platform supplies the configured default embedding model ID when creating a knowledge base. Docs continues to execute that approved configuration and does not regain a public model-management surface.
- Pass the two built-in model provider keys only to the private knowledge container and rerun a fresh upload through the public Platform API.
- The local DNS proxy resolves external model hosts into `198.18.0.0/15`. Use the existing deployment-owned `SSRF_WHITELIST_EXTRA` with the two exact built-in provider hostnames; do not weaken the default private-network block.
- Real ingestion now completes and writes one vector index. Search retrieval found the chunk, but Platform sent the unsupported `top_k` field instead of the typed Docs field `match_count`; fix that BFF contract and rerun search.

## Final Outcome

Local implementation and acceptance are complete. Docs exposes only authenticated `/internal/v1` knowledge execution routes, maps Platform organization IDs to private tenant workspaces, and has no shipped Vue frontend or public human authentication. Full Go tests/build and fresh end-to-end ingestion, search, download, ACL, and branding checks pass; production rollout remains separate.

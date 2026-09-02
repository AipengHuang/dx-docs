# T11 Project file classification

## Objective

Extend the existing post-parse auto-tag task with a typed, inspectable classification result and a Platform-only tag correction route.

## Implementation steps

- [x] Keep the existing asynchronous auto-tag pipeline and configured DeepSeek model selection.
- [x] Persist candidate confidence and evidence without changing access rules.
- [x] Mark uncertain, unsupported, incomplete, or malformed results as pending.
- [x] Classify from parsed text or OCR content, never from the file name alone.
- [x] Expose stable category provisioning and tag replacement through Platform-only internal routes.
- [x] Preserve manually attached tags on later parses.

## Affected areas

- `internal/application/service/knowledge_auto_tag.go`
- classification metadata helpers and focused tests
- internal knowledge routes

## Verification

- Run focused Go tests for classification and internal routing.
- Run `gofmt` and `go test ./...`.
- Run `ai-code-check` and address material findings.

## Progress

Complete. The existing post-parse worker remains the single classifier. Category mode emits one typed decision, attaches only one evidence-backed high-confidence category, and persists a pending result when content or evidence is missing. Platform correction replaces the category relation through the already scoped knowledge service.

## Final outcome

Implementation and verification are complete. `gofmt` reported no pending files and `go test ./...` passed across the repository. Focused tests cover confirmed, uncertain, malformed, unsupported, filename-only, manual-preservation, stable-tag, and idempotent-provision paths.

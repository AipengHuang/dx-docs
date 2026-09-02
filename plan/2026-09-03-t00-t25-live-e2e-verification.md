# T00-T25 live E2E verification

## Objective

Verify the Knowledge service through the running local topology and the Platform authorization boundary.

## Steps

- [ ] Verify live health, service authentication, upload/processing, search, citations, connector registry, and authorization denial.
- [ ] Verify persisted Knowledge state and Platform audit correlation.
- [ ] Record results in the shared T00-T25 evidence matrix.
- [ ] Fix and re-run only confirmed defects.

## Affected areas

- Knowledge internal API, DocReader, PostgreSQL, and Platform Knowledge gateway.

## Verification

- Real HTTP requests through Platform plus direct internal denial checks.
- Container health and database evidence.

## Progress

- [ ] Knowledge evidence captured.
- [ ] Authorization evidence captured.
- [ ] Failures resolved or explicitly reported.

## Final outcome

In progress. The current live Knowledge image still rejects otherwise valid execution grants, so Knowledge cases remain open until the current source image is rebuilt and re-tested.

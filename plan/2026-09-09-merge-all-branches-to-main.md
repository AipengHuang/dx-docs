# Merge all branches into main

## Objective

Integrate all local and origin branches for dx-docs into main, as explicitly
requested. Retain the recorded remote main content on conflicts.

## Steps

1. Inventory branches and clean worktrees; fetch origin.
2. Merge historical features and backup tips before subsequent reverts, then current fixes.
3. Record conflict paths and the remote-main resolution used.
4. Review the resulting code and run appropriate tests and ai-code-check.
5. Push main without force and verify every input tip is an ancestor of remote main.

## Verification

The starting origin/main is `a00ab7996a0ff524c16e6fa051a37390203fa20c`. The input inventory and detailed
merge results are recorded in the parent workspace under
`.local/merge-all-20260909/`. Check code-tree differences, current tests and
ancestry; preserve existing branch names and all working-tree data.

## Progress

- [x] Inventory and fetch.
- [ ] Merge candidates and conflict resolution.
- [ ] Code review and verification.
- [ ] Publish and prove branch inclusion.

## Outcome

In progress.

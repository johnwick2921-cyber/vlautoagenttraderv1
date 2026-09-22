# Planner chart frontend deployment — September 22, 2026

Owner authorized fix and deployment in this thread. PR #179 is merged.

- Source revision: `460d551f9758f9eb0cd72232ec2d09b95c567d8e`.
- Verified over HTTP at `2026-09-22T21:05:30.400947+00:00`.
- Served entry asset: `assets/index-DtrJkL6e.js`.
- Served SHA-256: `92157a4a6823e537a34a2bb933a2d43f0c0a0de52d8b5ce52bcc302b7657ba71`; byte-matches the clean-clone build.
- Running Go remains `a2bac00dc2ab06809c414f5049ddba3e77610f0c`, PID 75597; health `ok`.
- AddOn and trading settings were not changed. No process restart occurred.

## Validation

Clean clone: `/tmp/planner-chart-release-20260922/nofx`, at the merged HEAD.
`go test ./...` passed. Frontend: 77 files / 495 tests passed.
TypeScript/Vite production build passed. Regression tests first reproduced
the discarded-response defect before applying the one-line fix.
The chart endpoint requires authentication (401 to the unauthenticated audit);
no claim is made of an authenticated browser visual inspection.

## Release method

The existing Go static server reads `web/dist` per request. Installed immutable
assets first, then atomically replaced index.html. Old assets were retained.
GUIDE_BUILT_REV and deploy/RELEASE continue to identify the unchanged Go binary;
this receipt separately identifies the frontend source revision. Rollback copy:
`/tmp/planner-chart-release-20260922/previous-dist`.

Pre-existing import formatting in store/armed_orders.go and untracked .cgcignore
were preserved in stash `10b9cd311432a7d81577845bb35e808da9ea896a` for restoration
after the clean main-tree fast-forward. Neither is part of this change.

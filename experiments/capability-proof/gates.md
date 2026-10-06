# Quality gates for the proof checkpoint

Capability result and repository quality are separate. A passing test reproduces C5 FAIL; it does not make the capability pass.

The integrated baseline `75031b8eca8161b0a247414741828bfb2e1d9069` passed [CI 37391871008](https://github.com/Glacius-Labs/Markitect/actions/runs/37391871008): Ubuntu and Windows quality jobs plus the existing main-source artifact job. Historical PR #80 candidate `6212c157adc5866b2c3f65027911233495f5ff79` passed [CI 37390711975](https://github.com/Glacius-Labs/Markitect/actions/runs/37390711975). Neither is a release or a new capability result.

The witness was frozen before execution. [Run records](runs/windows-2/run.json) bind its command, exact harness revision, platform, Go version, raw-log hash and FAIL interpretation. Preserve run 1 as well as repeated observations. Normal quality gates for the final candidate are independently reviewable in the integration PR's exact-head CI receipts; do not substitute baseline CI for those receipts.

Required checkpoint checks use Go 1.27.1:

```sh
go test ./...
go vet ./...
go build ./...
go mod verify
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect-check-architecture --repo .
go run ./cmd/markitect-check-artifacts --repo .
go run ./cmd/markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config
git diff --check
```

Normal CI additionally runs executable examples and packaged source/bootstrap/selective-adoption gates on supported Windows/Linux hosts. No release publication command is part of this proof. Full-gate logs remain quality evidence; they are not fresh C1–C13 trials.

The evidence manifest hashes retained raw capability observations. It excludes itself. Baseline/experiment source commits and fixture revisions are different identities and must not be collapsed.

# Government G2 mechanics example

This standalone, dependency-free Go module exercises a single prior-authorized inventory change. The initial `Release` implementation is intentionally unsafe and its overflow test fails. The configured executor returns real changed Go source and test bytes; the Host applies those bytes only in its isolated candidate workspace. A later verifier process inspects the final source and test bytes supplied to its invocation and emits observations for each requested subject. Two separately invoked Ressort slots must then return explicit candidate- and evidence-bound responses. One fixture mode records `assent-unaffected` for the maintainability Ressort.

The model contains one Requirement, one root Area, one prior Mandate allowing `implement`, `review`, and `amend-model`, and exactly two Cabinet Ressorts. The inventory root observes the fixture's `government.yaml`, `order.yaml`, `go.mod`, runner, and inventory files. `README.md` is excluded with an explicit reason. Every included file has one declared writer; the Go module manifest, implementation, and test are explicitly related to the Requirement. The Order's Constitution digest is filled after `markitect government --action inspect` computes the selected model digest.

## Runner protocol

`runner/main.go` is a standard-library-only external process. It reads one `agentexec.Invocation` JSON object from stdin and echoes that invocation's API version, run ID, nonce, role, and input digest in its `agentexec.Response`. All four required response arrays are always present. The runtime config gives each process an explicit mode argument:

| Mode | Mechanism exercised |
| --- | --- |
| `executor` | Proposes changed `inventory/reservation.go` and `inventory/reservation_test.go` bytes. |
| `verifier` | Checks the exact final artifact bytes supplied in the request and reports one observation per `scopeIds` entry. |
| `assent` | Returns a fresh explicit assent observation bound to the candidate ID, evidence ID, and round in the current review context. |
| `assent-unaffected` | Returns explicit assent with a reason that this Ressort's concern is unaffected, bound to the same current review values. |
| `objection` | Returns a failed review and an objection detail; it cannot satisfy unanimous assent. |
| `stale-vote` | Returns a passed response whose vote detail binds a deliberately foreign evidence ID. |
| `missing-review` | Returns a passed verifier response with only the first requested scope observation. |
| `mutate-candidate` | Encodes a complete valid assent response, then appends bytes to `workspace/inventory/reservation.go` from the invocation context. |
| `incomplete` | Returns no review observations, exercising the missing-evidence path. |
| `wrong-binding` | Returns an otherwise shaped response with a changed nonce so the protocol layer must reject it. |

Each configured actor is started as a new process with a distinct slot and request context. The fault-injection modes are negative mechanics tests: stale evidence, incomplete scope coverage, and bytes changed after assent must each prevent acceptance or promotion. The executable can be shared by deterministic slots because this is a mechanism fixture; it does not demonstrate independent authorship or actual human judgment. The verifier's simple source checks and configured `go test ./inventory -count=1` check are deterministic evidence only. They do not establish broader semantic correctness or real model review.

## Running the native Host trial

Build Markitect from the source checkout to an absolute path outside that checkout, then run the one-command trial. The script requires an existing absolute output directory outside the checkout and the source-built executable path:

```powershell
go build -o "$env:TEMP\markitect-g2.exe" ./cmd/markitect
.\examples\government-g2\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g2.exe"
```

The trial copies an explicit fixture file list, including `trial.ps1`, into a new unique Git repository under the output directory. It commits the prior baseline, records its full commit ID, and initializes `refs/markitect/government/active/example` to that commit. The user's source branch is never used as the candidate branch. The script builds the fixture runner outside the observed copy, writes its runtime JSON outside the copy, and invokes the native command with both the absolute runtime path and required `--write` flag:

```text
markitect government --action run --repo <fixture-copy> --config government.yaml --order order.yaml --runtime <absolute-runtime.json> --write
```

The runtime sets `timeoutSeconds` to 1800, uses the `go test ./inventory -count=1` check with a 60-second check limit, assigns separate process slots for executor, verifier, and both Ressorts, and pins the expected base and managed active ref. The copy includes `README.md` and `trial.ps1`; both are explicitly excluded from the observed candidate boundary. The order's `activeConstitution` must equal the digest from the selected prior Constitution. Runtime, state, temporary workspaces, runner executable, and CLI stdout/stderr remain outside the copied repository.

The script saves native stdout as JSON and stderr separately. It writes `summary.json` with the source checkout HEAD and dirty status, native exit code, report status/stage, effective 1800-second runtime limit, result and runtime JSON SHA-256 values, Markitect and runner binary SHA-256 values, base, final checked-out HEAD, managed active-ref target, and promotion status. Exit 0 requires a parsed `accepted-scoped`/`complete` report with the 1800-second limit, successful promotion, active ref equal to the candidate commit, and checked-out HEAD unchanged at the baseline. A failed or incomplete run stays failed and retains its output for inspection.

To exercise the fault-injection modes, assign `missing-review` to the verifier slot to omit scope coverage. Assign `objection`, `stale-vote`, `mutate-candidate`, `incomplete`, or `wrong-binding` to a Ressort slot to exercise vote rejection paths. `mutate-candidate` requires the Host to supply its absolute candidate workspace in the invocation context; the fixture writes only the fixed `inventory/reservation.go` path. A stale active base remains a Host-level rejection case.

The runner modes are deliberately configured test programs, not a Scientist adapter. The public adapter v1.1 request/result wrapper and an actual Codex process are separate integration work. This fixture proves that the native Host can call literal external programs, bind their outputs to invocation receipts, require complete review coverage, and gate promotion; it does not prove autonomous engineering quality, live model review, provider authentication, or a production Git transaction guarantee.

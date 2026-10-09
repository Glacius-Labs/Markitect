# P03 Git workspace implementation handoff — 9 October 2026

P03 service implementation is complete at **f31edf9a374b52d64f425b5b92e65476c4507ee1**, on `codex/product-workspaces-20261009`. This is a development implementation with real local Git/tool fixture evidence, not native agent acceptance, a release, Main integration or human product acceptance. Product Integration owns wiring, candidate conversion, durable orchestration and the central queue.

The independently verified starting point was `1495e1be7b0046711531fa42c8407fba67b8e814`. Integration's shared file/directory-replacement contract fix `bb79936fb7676cf91c553510bc469fefd5f19f60` was cherry-picked as `9ce13ad361654291b5c45e9e8c9b2bd622052772` before the P03 implementation commit. P03 did not independently edit the shared contracts. Integration can cherry-pick **f31edf9a374b52d64f425b5b92e65476c4507ee1** after its existing contract fix; the cherry-picked prerequisite does not need to be applied twice. This handoff is a separate documentation commit whose final remote SHA is reported in the material callback.

## Concrete API

```go
func NewGitService(storageRoot string, limits Limits) (*GitService, error)
func InspectRepository(ctx context.Context, root, base string) (Binding, error)
type Binding struct { OverlayDigest, InventoryDigest string }
func CandidateOverlayDigest(changes []Change) (string, error)
func (*GitService) PrepareCandidate(ctx context.Context, request Request,
    candidateOverlay []Change, candidateOverlayDigest string) (Handle, error)
```

`GitService` implements the existing `Service.Prepare/Harvest/Close`. `Prepare` calls `PrepareCandidate` with no parent overlay. `InspectRepository` supplies the caller's required source-WIP `Request.OverlayDigest`. Caller repository identity and task identity remain explicit inputs. Source top-level root and full selected SHA are checked against the actual Git checkout; source HEAD must equal the selected base. Shallow sources fail explicitly.

`PrepareCandidate` validates the canonical overlay digest, finite configured limits and actual operation existence against the captured source WIP inventory. A parent overlay can provide foreign READ context without expanding the child `AllowedPaths` or weakening `ExcludedPaths`. Source WIP freshness and materialized candidate baseline are separate: `Handle.BaseDigest` hashes the actual initial candidate inventory including parent output; the source inventory digest is retained separately in private service state. The full immutable handle binds the state; forged/unknown handles fail.

Storage must be an absolute directory outside the physical source repository. Prepare creates a private local clone with `--no-hardlinks --dissociate --no-checkout`, real source history and branch refs. It detaches HEAD and sets the index with Git data operations, then materializes exact source/overlay bytes. It removes the origin configuration while preserving its cloned refs. No original checkout, index, history, remote or WIP is changed. There is no configured push remote. Service Git acquisition ignores ambient Git redirection/system/global configuration and disables fsmonitor; it does not execute checkout filters/hooks. Normal tools invoked by a runtime in CWD retain that runtime's existing user policy; the service does not create an OS sandbox or elevate permissions.

## Inventory and delta truth

- Source capture observes regular files throughout the repository, including ignored and uncommitted WIP and canonical `.markitect` inputs. It skips Git metadata and exactly `.markitect/runs`, the Host's mutable private logs/receipts/candidate/check-record subtree. These records remain untouched in the adopter and are not copied into task context. Other `.markitect` state, model and runtime files are not exempt. All foreign/excluded write scopes stay observed.
- Candidate observation skips only Git metadata. Any candidate-created `.markitect/runs` records are observed and rejected as control writes. Empty allowed scope permits an unchanged candidate, and rejects any data write. Both rename paths must satisfy scope and exclusions.
- Inventory reads use confined filesystem access, reject links, Git symlink entries, submodules/gitlinks, special files, unsafe paths and case aliases. Unsupported links/submodules fail visibly rather than supplying incomplete context. On Windows executable modes are taken from the real Git index; new executable outputs must be staged with their mode (for example `git add` and `git update-index --chmod=+x`). Unix executable bits are observed from filesystem mode.
- Add/modify/delete/rename changes carry exact bytes and Git modes. Git similarity detection preserves ordinary edited/mode-changed rename provenance; fully rewritten unrelated moves remain delete/add, consistent with what Git can establish. Binary content is never decoded as text. Final file/directory transformations are supported by the integration-owned contract correction.
- Writers must be stopped before Harvest. It verifies file identity/size/mode/mtime around bounded reads, recaptures the final inventory, rechecks source freshness, checks candidate Git identity/history, and then returns normalized observed changes. These checks detect ordinary races; they do not claim OS isolation against an adversarial concurrent writer.
- Full read context has finite ceilings: 100,000 files, 256 MiB per file, 1 GiB inventory bytes. Public overlay digest has ceilings of 10,000 changes, 64 MiB per file and 256 MiB total; PrepareCandidate applies the configured service limits before cloning content. Harvest enforces changed-content/count limits during capture, then delta normalization; rename scratch is separately capped at 256 MiB. Oversize inputs produce explicit errors, never partial accepted inventories.
- Failed Prepare removes only its newly allocated private clone. Close validates stored ownership and removes only that candidate; canceled Close can be retried. Separate candidates remain usable. Unknown handles after service/process replacement are rejected and their files/evidence remain intact. Automatic cross-process adoption/recovery of an existing workspace is not implemented; Integration must preserve and reconcile uncertain evidence rather than blindly replay or delete it.

## Files and ownership handoff

P03-owned new implementation: `internal/host/projectworkspace/git_service.go`, `git_inventory.go`, `git_overlay.go`, `git_renames.go`.

P03-owned new tests: `git_service_test.go`, `git_overlay_test.go`, `git_hardening_test.go`, `git_recovery_test.go`.

P03-owned record: this file, `docs/work-items/product-readiness/P03-workspaces-handoff-20261009.md`.

Product Integration must register these nine paths in its canonical artifact accounting and maintain central README/backlog statuses. The local managed-artifact check currently fails solely because the new P03 files have not yet been registered; P03 did not mutate that shared ownership file. This is an explicit integration task, not a passing artifact gate. Existing shared DTOs, agentexec, projectrun, Apply conversion and runtime composition remain Integration-owned.

## Checks and independent review

On committed implementation SHA f31edf9a, `go test ./internal/host/projectworkspace ./internal/tooling/architecture -count=1 -timeout=2m` passed: projectworkspace 41.549 seconds, architecture 0.713 seconds. Earlier focused package runs passed while implementing concrete fixes; no full unchanged repository test suite was rerun. All-package compilation (`go test ./... -run '^$' -count=1 -timeout=3m`) passed on the implementation working candidate before the last operational-subtree/history-copy adjustments; final package execution compiles and verifies those changes.

`go vet ./internal/host/projectworkspace` and staged `git diff --check` passed on the final implementation. Fixed-source `markitect check --repo . --revision f31edf9a374b52d64f425b5b92e65476c4507ee1` passed. Managed-artifact accounting remains pending registration as described above.

A bounded independent developer implemented the real-Git lifecycle tests in its exclusive new test file. A separate read-only developer review identified and closed overlay resource bounds, edited rename provenance, silent gitlink omission, Git configuration/filter execution, inventory allocation limits and capture races. Its final review checked the exact working implementation subsequently committed as f31edf9a, including the operational-record and branch-ref boundary; it reported no remaining actionable defect in that scope. Review itself ran no tests and started no native product roles.

Fixtures exercise real Git log/blame/two-commit history/branch refs, source staged/untracked/ignored WIP preservation, native `go test` in candidate CWD with cross-file references, binary output, executable modes, edited renames, add/edit/delete, file-directory replacement, foreign/control writes, source staleness, parent overlay semantics, limits, separate candidates, cleanup and retained unknown evidence. These are mechanism/tool checks, not autonomous semantic/productivity evidence. The Windows symlink test was SKIPPED because this host lacks symlink privilege (confirmed by a focused verbose replay); its platform behavior is therefore not established by this local run. Indexed gitlink rejection ran and passed. Supported-platform hosted CI remains separate.

No native acceptance jobs or role-start requests were consumed, no Main/PR89 merge was performed, and no release/tag/package was published. At source publication, `gh run list --branch codex/product-workspaces-20261009 --limit 5` returned no hosted runs; this is NOT RUN, not a passing hosted gate. Full integrated product readiness, hosted CI and human acceptance are not established by this handoff.

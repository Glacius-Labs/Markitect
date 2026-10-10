# Byte binding and Integrator follow-up, 2026-10-10

Direct human request: explain the rejected revision-bound exploration preview,
assess excessive strictness, and investigate further problems. Product source
observed at `e10ad79a8fb238f016fa9026796fc38df8a35a82`, in Product Integration's
checkout. Implementation-only research evidence; no product-intent change.
No Product file edits, model/provider starts, installation or global changes.
This supplements the [earlier assessment](integrator-assessment-20261010.md).

## Actual README rejection

The protection should prevent publishing a preview after its relevant inputs
have changed. It does not require finished implementation bytes to equal their
original source and does not measure semantic correctness. In this incident the
refused operation was storing exploration metadata, not an OS denial of a README
edit. `projectapp/exploration.go:237-333` loads a selected snapshot and hashes its
files; `projectexplore/store.go:317-338` compares them with captured worktree bytes.

The actual acceptance adopter is
`C:/Users/Consiliari/AppData/Local/Temp/markitect-a01-3c5c733388014645b0ca178338078b6a/repo`.
Its HEAD is `eaa11e9969db66e6c1a5cd80949cfb757074ce54`. Read-only byte comparison
and an independent repeat establish:

| README source | Size | SHA-256 | Line endings |
|---|---:|---|---|
| Git blob | 78 | `d2286628c200f45827b9b5407aff1a5a0f30dabe9eacc781f95e9a00b5dcbffc` | two LF |
| Worktree | 80 | `12b6563d10ab227791722774d264084d20653a24d3a891347b1c1b8e12e23388` | two CRLF |

No BOM; bytes match after CRLF-to-LF conversion. Git reports README clean.
The adopter has no `.gitattributes`; `git check-attr -a -- README.md` is empty;
system `core.autocrlf=true` applies. The product source repository's LF attributes
do not govern this adopter. Thus the rejection is a confirmed clean-checkout
false positive. The Integrator's working-tree-selected retry avoids this byte
domain mismatch, but leaves the explicit-revision product path defective.

**Repair:** compare fixed revision inputs with Git-filter-aware source equality,
then capture actual worktree bytes as the preview baseline and require their
exact equality at publication. Preserve source revision, HEAD, branch, repository
identity, expected plan digest and target-record CAS. Planning already implements
Git-aware equality in `projectrun/plan.go:641-704`. Test both a normal CRLF checkout
and a real edit between preview and write. Disabling `VerifyBasis` or blindly
normalizing arbitrary artifact bytes would lose a different safety property.

## Binding breadth

The "model-only" exploration includes every `project.Snapshot.Files` entry in
`BasisFiles` (`projectapp/exploration.go:310-323`), even though it computes a scoped
file selection separately. Full coverage therefore includes README and ordinary
implementation inventory. Readiness's explicit basis uses manifest, runtime and
model files (`projectrun/exploration.go:149-153`). Snapshot-wide identity can be
intentional; an unrelated file edit blocking model-only metadata is a usability
and contract concern, rather than proof every exact-byte guard is excessive.
Define the relevant dependency closure explicitly before narrowing this guard.
The old exploration record can be preserved while a fresh preview uses the actual
current basis; historical evidence does not require forcing a clean checkout to
have Git blob line endings.

## Additional findings and current run

Actual15 plan `3c41c26278ec1b579077ebe03b667f6d`, state revision 22, is blocked.
Root work/review passed; Docs review is uncertain; Source remains invoking in
that recorded state. The genuine helper reservation is completed with an
`applied-and-closed` delivery. The Docs reviewer failed opening its private event
journal for append with a Windows sharing violation. The observed journal is
`private/codex-app-server/57b30085daf783d5da7ef6d1f6a365ab322e5bbe6dfb0c27a540fcdac38ed7c8/77cb625e19e00dd1fa58da9ca8d780c7/events.jsonl`.
The subsequent Resume was rejected. No successful Verify/Apply/recovery follows
from this evidence.

1. **Recovery loses its diagnostic cause.** `run.go:160-176` discards the non-nil
   error from `recoverPendingNativeReview`, persists a generic failed-recovery
   finding and returns a generic new-plan error. `resume_review.go` returns the
   `invokeReviewer` error, which may already be a safe recovery-error wrapper;
   this outer branch discards that diagnosis. The earlier journal-append failure
   remains visible, but does not establish the separate failed-Resume cause.
   Retain a safely bounded cause in diagnostics
   and the returned error; preserve recovery identity checks and no-replay rules.

2. **Helper cleanup has no reconciliation route.** `helper.go:446-490` can apply
   a valid child delta, fail child `Close`, persist `cleanup-pending`, and leave
   the durable reservation `unknown`. Resume dispatch in `run.go:149-203` covers
   pending Manager/reviewer turns; it does not reconcile this Helper transaction.
   `obligations.go:347-362` then blocks closure. Existing helper tests preserve
   this state but do not exercise a subsequent successful recovery. Child native
   handles/events are independently journaled by `TransportInvoker.Run:76-82`;
   clearing parent callbacks does not discard them. Add bounded cleanup retry
   and delivery reconciliation against the original journal/receipt/delta,
   without applying the delta or starting the child turn again. This is a static
   failure-path gap, not the observed cause of Actual15.

3. **Live auditing can interfere on Windows.** A provider-free probe using only
   a newly created Scientist scratch file confirmed an active .NET `ReadLines`
   enumerator prevents concurrent append (sharing violation `0x80070020`), while
   a reader with `FileShare.ReadWrite | FileShare.Delete` permits it. The owned
   probe file/directory were removed. This does not identify the actual lock
   owner. Product `native_journal.go:244-260` already serializes in-instance event
   appends; `:279` opens the append handle. The A01 observer is excluded for this
   specific Docs lock: it never reached Root integration, and its event reader
   runs only after process exit on the Root journal. Integrator acknowledged a
   possible restrictive live audit read. Use explicitly shared short reads of
   active logs and parse copied bytes after releasing the handle. Preserve the
   actual append failure; no foreign process termination or permission changes.

Two independent source workstreams reviewed the byte-domain and helper/recovery
paths. Findings and concrete repair directions were sent to Integrator. Product
changes and any next acceptance execution remain Product-owned. The CRLF defect
is demonstrated on the actual adopter; the cleanup gap is source-backed; the
specific Docs journal lock owner remains unknown.

State revision 22 whole-file SHA-256:
`cfefdcabeff7ce9100ab2fc6de2251af635bf7b32659ada1728262c90e456af4`.
Observer `741392ec2c04437db31d29f6e821d3c6.observer.json` whole-file SHA-256:
`29d7f21a26372824ddd6310ff7e1cd89a1846e8c12e45c8dadb7fbe02c7de149`.
Both retained in the Product-owned acceptance root above; raw logs were not
copied into this report. These observations establish no method comparison,
Main/release readiness or human acceptance.

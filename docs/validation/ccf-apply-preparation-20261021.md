# CCF-20261021-v1: source and control preparation

This is preparation for the owner's new bounded series, not a provider outcome.
The integrated base is `445cea2ab1e7e4fbc0de3837b5038a9c34ee0425` (#590).
The writer uses isolated `fix/apply-evidence-acceptance`; the shared checkout's
untracked work was preserved. [The handoff](../development/codex-apply-conformance-handoff-20261021.md)
and [accepted workflow](../design/canonical-model/manager-apply-workflow.md#evidence-backed-acceptance-after-the-codex-retest)
own the assignment. The previous [three-attempt record](apply-codex-conformance-20261021.md)
and its exhausted ledger remain unchanged.

## Mechanism and limits

The existing common Pass now requires material `conditions` in Implement/artifact
review coverage, typed public behavior/static evidence, and explicit parent integration
in the same basis/result. Root cannot evade integration through empty own Definitions.
Receipts bind the submitter, obligation, stand, probe bytes and invocation; a public
claim needs the matching declared boundary. One receipt cannot count again for another
claim, including integration. Source evidence remains valid for genuinely static claims.
Conditions and boundary labels are declarations for semantic review, not machine proof
that every natural-language condition or public entry was actually covered. Parents and
fresh reviewers must inspect assertions and challenge omissions. Missing required
execution stays open. Fresh Verify records its own conditions/counterexamples before
assessing author coverage, including valid inputs wrongly refused. Extract keeps fixed
artifacts/documented intent as authority; the candidate model is its work product.

Implement satisfaction is a checked submission to independent Verify. Immutable fresh
Verify, adverse receipts and independent disproof, findings, strict Model/File accounting,
closed Runs, ownership, per-Turn cessation and guarded publication/read-back remain.
No consumed Cockpit event key was removed or renamed; added envelope/receipt/budget
fields do not alter its current reader contract or commission UI work.

The smallest budget seam is optional `workDeadline` in the invocation and shared Run
budget, bound alongside the effective whole-run deadline. An active non-Verify Pass
is canceled there and ends Incomplete; unfinished work never starts Verify. Setup does
not choose a reserve. CCF fixes work at minute 29, the provider/product deadline at
minute 41, the Apply process at minute 41:30 and the complete attempt at minute 45.
Thus the review reserve is 12 minutes, with 3 full minutes after the Apply process ceiling for observed stop/cleanup before
the wrapper soft stop at minute 44:30, then 15 seconds of kill grace and 15 seconds
of margin to the grant ceiling. All deadlines and elapsed charges are anchored to
wrapper-child process entry (UTC and monotonic), including prelude checks; the ledger
separately records when the reservation became durable, before execution effects. A correction still uses the original work
cutoff. Limits are 240 Agent turns, 8 Passes, 4 minutes per Turn and the existing
12 probes per Turn; check/profile limits also remain bound. These are finite ceilings,
not completion predictions. Unknown cessation/cleanup blocks all later starts.

## Independent control adjudication

Earlier offline review was partly conditional. The initial new review overclassified
both disputed edits as proven defects; a targeted challenge corrected that conclusion.
The old result is not overwritten.

- **Large IDs:** [Depot inputs](../../experiments/apply-trials/depot/snapshots/d1/model/depot/inputs.yaml)
  gives positive decimal order IDs no authored maximum; [Order](../../experiments/apply-trials/depot/snapshots/d1/model/orders/order.yaml)
  requires first ID 1 and every later ID the next integer. A conforming history with a
  present ID above 18 digits is not feasible to construct for this bounded test.
  The actual CLI refuses an absent `1000000000000000001` before and after the source
  alignment; no exact error text or private-parser test establishes a historical public
  defect. The CCF-only overlay removes the explicit 18-digit policy cap for the unbounded
  field, retaining public refusal when Python's numeric representation cannot process
  input. This is conservative source alignment, not a demonstrated reachable historical
  CLI failure or proof of arbitrary-size persisted IDs. Its tests claim only public
  refusal and non-write for absent values. The historical ambiguity remains recorded.
- **Cross-section state:** [Availability](../../experiments/apply-trials/depot/snapshots/d1/model/orders/availability.yaml)
  requires `available = onHand - open reservations`, never negative.
  [Storage](../../experiments/apply-trials/depot/snapshots/d1/model/depot/storage.yaml)
  requires malformed existing data to fail through the public interface without overwrite;
  internal JSON organization is implementation freedom. A real baseline CLI sequence
  `add-item BOL-006`, receive 3, place 2 builds a valid state. Mutating only the observed
  stored stock from 3 to 1 creates a contradictory malformed state; baseline `items`
  exited 0 with `onHand:1,reserved:2,available:-1` and preserved those bytes. This is not
  a normally reachable state. Interpreting this contradiction as malformed is an explicit
  inference from both source contracts. The CCF-only overlay calls the actual Orders
  availability rule at load and refuses before queries or changes. Its reference-specific
  regression mutates its own observed representation and checks exit 2/error envelope and
  unchanged bytes, not exact errors, serialization or a portable storage schema.

`D-CF` and `D-CFH` share exactly the same model, registry and corrected unattractive
reference (`UNATTRACTIVE_EDITS + CCF_REFERENCE_EDITS`). Only `D-CFH` injects existing
`N1_HYPHEN`: remove general name-token attachment and its one self-test. The external
name discriminator covers several equivalent option-like valid values. Literal trial
answers/hidden expectations never enter Manager/reviewer prompts. `CCF-P` preserves,
`CCF-R` repairs, and `CCF-G` uses unchanged `S-QV`/A4-gap model bytes with a new finite
budget. Root must confirm its material gap during Assessment, without Implement,
publication or an external answer. Historical D-U/D-UH and A1/A2 v2 remain intact.

Independent materialization/CLI checks passed both new reference-specific tests in
both packets. The external d1 oracle passed D-CF 22/22 and rejected D-CFH on exactly
`item-inputs` and `name-hyphens` (20/22); D-CF accepted all five tested option-like names.
D-CFH rejected the three option tokens and accepted the numeric-looking/single-hyphen
values, matching the scoped defect. Broad selfcheck/evalcheck runs were interrupted;
they supply no result or replacement for this focused evidence.

## Evidence and start gate

Provider-free protocol tests distinguish missing/weak evidence, declared public versus
static evidence, empty Root integration, cross-envelope receipt reuse, mandatory fresh
review conditions, active work cancellation and Verify's remaining window. Their Agents
and runner are deterministic fakes: they do not prove semantic model detection. Actual
fixture CLI/oracle checks are separate from those protocol controls and from any real
provider result. Existing safety regressions retain their original scope.

Independent semantic review caught receipt reuse across obligation/integration and a
mandatory Apply-oriented prompt conflicting with Extract authority. Both were corrected
with distinguishing regressions. Complete-diff, controls and finite-runner review results,
final local checks and exact frozen revisions are recorded in the PR and frozen manifest.
The complete-diff review also caught missing non-Plan foundation-integration evidence,
a boundary omitted from the pre-probe event, Plan wording and a shortened cleanup
reserve; these were corrected. The CCF-only scorer now requires the complete fresh
six-Manager Verify and matches its reported tree and Pass identity to candidate/sealed
events. Run/Publisher owns binding that immutable tree to published bytes; this is an
event/report cross-check, not an independent tree rehash. Protocol controls reject
missing, wrong-Pass and mismatched-tree events. Both outcome and process must pass;
unpriced reported cost is retained separately and monetary cost stays unknown.
No hosted CI or release is claimed.

The new Linux wrapper [ccf_run.py](../../experiments/apply-trials/ccf_run.py) derives from
the prior bounded runner. It validates source/binary/kit/profile/control identities,
reserves durably before effect, runs only one of three fixed serial positions, preserves
failed evidence/candidates, and blocks following starts after unknown cleanup. It has no
retry/fallback or generic scheduling engine. New ledger starts empty. After durable reservation and before every provider start,
the wrapper repeats the zero-generation dedicated-login/model/effort/account/quota read,
requires at least the attempt ceiling of token validity and 25% remaining in the live
Codex primary bucket, observes metadata-container removal, and retains a hash-bound
readiness record with UTC and monotonic timing. That conservative stop threshold is
an operator limit, not a prediction that the remaining quota covers three attempts;
the coordinator still approves the observed quota reserve. The source-bound
[metadata reader](../../experiments/apply-trials/ccf_metadata.py) sends initialize,
model/list, account/read and account/rateLimits/read only, never thread/start or turn/start.
The latest live dedicated-volume check at 2026-10-21 19:12 UTC reported ChatGPT Pro, the requested
`gpt-6.1-sol`/`high` catalog entry, 51% weekly quota used and access expiry
2026-10-27 13:04:13 UTC. It generated zero requests. A catalog is not model entitlement
proof ([official App Server documentation](https://developers.openai.com/siwc/token-sharing-open-source/codex-app-server));
no inference is hidden in preflight. Quota/expiry and host slot must be refreshed before
starts; quota availability does not guarantee measured cost or completion. Monetary cost
is unknown. Only existing dedicated `mk-codex-runtime-auth` is used; no refresh token
reaches sessions, and no desktop login/API-key/model fallback is allowed.

The initial host observation had no Markitect containers, with three existing WM demo
services running; this is reported explicitly for the coordinator's host-slot decision.
The provider gate remains closed until the delivering chat reports its exact reviewed
manifest and `CCF TESTBEREIT`, then the coordinator gives the one source-bound release.
No provider starts are authorized by this record itself. After release, at most three
attempts in order CCF-P, CCF-R, CCF-G and 135 summed minutes, including failures/generative
preflights; no build correction in the frozen series, fourth attempt or further wave.

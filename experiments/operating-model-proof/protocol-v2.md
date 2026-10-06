# Real-agent operating-model proof protocol v2

Protocol ID: operating-model-proof/v2. Frozen before C9 execution.

This version preserves all v1 claims, measurements, authority rules, historical outcomes and C3/C4/C5/C8/C10 controls. It supersedes only v1's C9 fixture blocker with a separate source-bound recursive fixture and controlled negative materializations. No C9 result is claimed by the fixture, its unit tests, or this protocol.

## Frozen product and fixture revisions

- Markitect product source for the first v2 trial: ac759436649a5d239c1c9ba4c735ee479862c5d4, or a later full source SHA explicitly frozen before invocation.
- C9 public fixture source: 4958a80b4fbbb129a2f276e0ae918018d0f70e35. The fixture is examples/recursive-assurance at that commit. Its checked-in tests bind canonical loading to full HEAD and test pins, typed scopes, policies, target roots, and required checks through actual preparation. Those are setup evidence only.
- The fixture defines Orders and Billing leaves, Checkout composition over both, and Commerce composition over two Checkout results plus Billing's shared rule. Public APIs and exception behavior are explicit in target ProjectionPolicies; target filenames remain Executor-chosen. Use the corrected source above so an API-name mismatch is not mislabeled as an AI semantic miss.
- Before execution, create and record a private fixture Git clone with a clean committed tree and full base/candidate/evidence IDs. Record the exact source digests, Module pins, all four scopes and their checks, runtime bindings and CLI build receipt. A later product/fixture change requires a new protocol version or an explicit pre-invocation freeze amendment.

## C9 real-agent positive path

Run the accepted dependency order: fresh real Executor produces candidate bytes for Orders and Billing; each is reviewed, applied and checked; then fresh Executors produce Checkout and Commerce candidates using actual child candidate bytes in their bounded context. Each parent gets its own fixed check and a fresh independent Verifier. Use the controller's dependency-ordered execution path and truthful child candidate staging. Record a real ProjectionRecord and VerificationResult for each scope; the Commerce root is complete only when its own check and independent verification pass. Child PASS alone is never root PASS.

The repository preparation test, a passing fixed check, and a real Executor receipt each support only their own bounded claims. No owner has accepted the fixture or output.

## C9 controlled recursive failure controls

The negative controls test assurance composition with deliberately incorrect parent-owned implementation bytes while all relevant child implementations remain correct. These are controlled non-AI materializations, not outputs attributed to an Executor:

1. Checkout control: preserve passing Orders and Billing materializations. Supply a frozen Checkout candidate that either drops the accepted Orders total when invoking Billing or bypasses invalid Orders input. Run the ordinary exact canonical candidate plan and explicit Apply. Retain the real plan digest, Apply output, actual materialized-unverified ProjectionRecord, record-store append/CAS active selection, and object-only evidence revision. Then invoke controller-verify with a fresh independent Verifier and Checkout's fixed check. Expected: Orders PASS, Billing PASS, Checkout FAIL/incomplete, Commerce does not pass.
2. Commerce control: preserve passing Orders, Billing and Checkout. Supply a frozen Commerce candidate that omits one Checkout result, loses prior outstanding amounts, or bypasses Billing's shared overflow rule. Use exact canonical plan/apply and the same immutable evidence/ledger boundaries. Fresh verification must retain passing children while Commerce fails.

The deliberately wrong parent files are declared control inputs with their own content digests and source bindings. Do not replace or edit an Executor receipt, invent a ProjectionRecord/VerificationResult, or claim AI produced the control bytes. Any fixed-check-only observation without a real Verifier is PARTIAL for the independent-agent claim. The mutation mechanism is reported separately from Verifier effectiveness.

Use distinct fixture copies, run IDs and ledger roots for each control. Preserve failed/incomplete events. Never allow the controlled parent record to replace or falsify child records. If exact canonical plan/apply cannot bind the control bytes and produce an actual record/evidence revision, classify the control BLOCKED; do not hand-write a substitute receipt.

## CLI build receipt required for every v2 run

Supply an external build-receipt JSON with:
- sourceSha: the exact full Git commit used to build Markitect;
- buildCommand: the actual command string or argv array;
- exitCode: integer 0;
- binaryDigest: sha256 digest of the exact CLI executable passed to the driver.

The sourceSha must resolve to a full commit object in the Markitect repository, and binaryDigest must equal the selected CLI file bytes. The driver stores a digest of the receipt and a digest of the command; the original build receipt remains external. A source HEAD observed after the build is not a substitute for this receipt.

## Historical status and limits

The v1 C9 classification remains BLOCKED for the original operating-model fixture. The separate recursive fixture changes only the readiness of a future fresh C9 experiment. It does not change prior C9 NOT RUN results, the protocol-v1 C5 targeted-acquisition FAIL, or any other historical status. The existing baseline has no fresh C9 Executor/Verifier outcome. Versioned real runs are the only path to PASS/PARTIAL/FAIL evidence.

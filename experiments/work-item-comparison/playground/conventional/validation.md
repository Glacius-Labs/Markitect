# Conventional wrapper preparation validation

Source preparation on 2026-10-09, following the direct user's request to include
Conventional control paths alongside Markitect MCP/App Server work. Scope is the
Scientist's private research checkout, not a product integration or runtime release.
The original source checkpoint `df6db9cc846378769efc9b8596b2a9736efe0425` opened no actual allowance. The younger direct user instruction and verified finite order now authorize only the two Conventional pilot trajectories in the separate plan; generic examples remain disabled.

## Focused result

**46/46 PASS**, `19.701s` driver wall time, exit 0. The prior 41-fixture result remains in the original checkpoint.
The exact command, UTC interval, tested source byte hashes and the syntax-checked pilot controller hash and technical log
hash are in [offline-checks.json](offline-checks.json); output is retained in
[offline-checks.log](offline-checks.log). Source hashes did not change during the check.
The technical unittest log is normalized from CRLF to LF for Git; the captured
pre-normalization hash is retained in the record. Native wire bytes are unchanged.
Technical log SHA-256: `9b0822a5e6246380dfa06439ba71ff25d331d3ed8ade7c593be3d5d7049696a3`.

```powershell
C:/Python313/python.exe -B -m unittest discover -s experiments/work-item-comparison/playground/tests -p test_conventional*.py -v
```

Eighteen backend fixtures exercise scripted local Python children; twelve service
fixtures inject callbacks; nine MCP protocol fixtures use injected services; five
composition fixtures cover MCP → service → each backend, CLI waiting/Unicode,
disabled stdio, and version-specific legacy MCP results. These are developer
checks, not implementing study actors, native Codex, installed App Server, provider
availability, effective serving or product acceptance.

Coverage includes nested App Server terminal status and exact identities, CLI
completion without status/turn ID, session continuation and mismatch, early terminal
notifications, authoritative failed/unknown status, approval requests with no auto
acceptance, partial/unknown native usage, original byte preservation, missing
terminal/timeout/cancel uncertainty, inherited output pipes and unread stdin,
external-order/config/executable/source pins, finite outer attempts, persistent
ownership lock, cross-controller cancellation, and worker-start failure with no
native dispatch or consumed parent continuation. Native terminal completion keeps
functional assessment `NOT RUN` and human acceptance unestablished.

## Review and scope boundaries

Independent source review found protocol shape errors, unsupported old MCP output
fields, a pre-dispatch worker failure and raw-text-only evidence. Corrections are
covered by the final fixtures. Windows atomic status replacement retries only local
file I/O for at most two seconds; native requests are never automatically retried.

The backend owns only its direct child process. Descendant/native helper termination
is not guaranteed. Unconfirmed cancellation, lost protocol/controller ownership and
unknown native terminal disposition retain evidence and block automatic replay.
Approval/user-input requests are captured as blocked; interactive approval and
uncertain-session recovery editing are not implemented. Control-route semantics,
loaded context/Skills/tools/permissions/helpers and equivalence with the interactive
CLI remain unverified. Request/model/effort metadata is not effective serving proof.
Native aggregate usage, billable cost and active helper time remain unknown unless
later independently evidenced.

This wrapper's admission guard binds a config, external finite order, run identity
and exact executable/wrapper source bytes. It does not authenticate human authority
or fully verify the pair freeze. Case/setup/evaluator/tool/rights/context binding and
method-specific product costs remain owned by the common freeze/study contract.
Future actual windows must be frozen generously; the disabled example proposes
90 minutes per turn and 10 minutes per protocol request, not a reused grant.

Product Go/Markitect CLI gates and installed runtime checks were not run under the
direct preparation-only scope. No product implementation was edited/imported.
Lifecycle Git/file mechanics, public inputs and historical Luna Nest are unchanged;
their previous 18 checks were not repeated. The new study-contract SHA is bound in
current freeze and assessment templates. No old evidence was rescored or replaced.

The files are source-prepared and offline-checked. The finite pilot is separately authorized; native results are recorded separately. Transport parity, Markitect method effects, economic benefit and human acceptance remain unestablished.

Runtime setup checks cover typed workspace-write/approval/memory/helper overrides, rejection before dispatch, and legacy configs. Pilot approval=never and memories=false are declared deviations from an ordinary interactive CLI session. Owned stdio needs no external daemon. The controller reserves 15 minutes within each four-hour job for capture/assessment and uses a shared eight-hour deadline. An operator MCP observation error is recorded while the same native controller continues; unresolved ownership blocks the next trajectory.

The first real pilot also exposed a controller call-site error after authoritative native completion. The metadata argument is now keyword-only at the call site. Two new provider-free controller fixtures execute the full four-stage capture/assessment/freeze path and verify that observer failure records evidence without stopping the owned actor. No real trajectory was replayed. The separate continuation script admits only the original terminal first receipt and still-unused BF destination, preserves the original overall expiry, and rejects repeated reconciliation/second-case setup.

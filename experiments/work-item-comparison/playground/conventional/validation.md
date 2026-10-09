# Conventional wrapper preparation validation

Source preparation on 2026-10-09, following the direct user's request to include
Conventional control paths alongside Markitect MCP/App Server work. Scope is the
Scientist's private research checkout, not a product integration or runtime release.
No actual study allowance was opened.

## Focused result

**41/41 PASS**, `6.117s` driver wall time, exit 0.
The exact command, UTC interval, nine tested source byte hashes and technical log
hash are in [offline-checks.json](offline-checks.json); output is retained in
[offline-checks.log](offline-checks.log). Source hashes did not change during the check.
The technical unittest log is normalized from CRLF to LF for Git; the captured
pre-normalization hash is retained in the record. Native wire bytes are unchanged.
Technical log SHA-256: `ef9ea309e784f37b6e4b2602b527d6b704c3ddf8c4b9a2775329886c35297d81`.

```powershell
C:/Python313/python.exe -B -m unittest discover -s experiments/work-item-comparison/playground/tests -p test_conventional*.py -v
```

Seventeen backend fixtures exercise scripted local Python children; ten service
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

The files are source-prepared and offline-checked. Actual execution, transport
parity, method effects, economic benefit and human acceptance are **NOT RUN**.

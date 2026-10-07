# Proposed common measurable profile correction

Status: Coordinator decision requested, not adopted. `resource-proposal.json`
and its commonLimits stay byte-for-byte unchanged in this package. No approved
live grant exists. Current native usage has unknown providerRequests, so even a
correct public response with known tokens ends incomplete and blocks the next
reservation. Merely choosing one agent turn cannot repair that mismatch.

The smallest proposed change is a **new explicitly approved measurement-profile
version shared by all three arms**. Keep the existing wall, session, parallel,
human, token and repair numbers, plus the smoke's stricter one-session/180-second
scope and retrospective 10000-token threshold. Keep token totals null when
unreported; unknown token usage must still block a subsequent session. Retain raw
input/output usage and record threshold/parallel overshoot rather than describing
tokens as a hard in-flight cap.

Change only the unsupported observation requirements: provider-request counts
and provider-internal transport retries become explicitly unobservable/null
measurements, without a hard-cap claim. Zero wrapper retries remains enforceable.
For the selected pinned native CLI, unchanged provider transport defaults may be
permitted inside the finite process wall only by that explicit decision. Native
child/model work absent from receipts remains an observability gap; every Actor
the harness itself launches still needs a separate granted ledger reservation.
The legacy ProviderTurns 80/480 and maxTransportRetriesPerCall 1 cannot be cited
as verified provider caps under this proposal; they must be marked unsupported
in the new approved profile, never replaced by agent-turn counts.

If approved, a bounded follow-up source change must distinguish unknown tokens
from unknown provider requests in admission and bind the new profile version to
the same cumulative ledger without erasing prior attempts. That change is not
implemented here. Until it is reviewed/frozen, this dispatcher's strict unknown
provider-usage stop remains in force. Do not work around it with a new database,
new trial identity, reset, inferred counts or a mode flag.

Reproducibility uses the exact requested ID `gpt-6.1-sol`/high, its authenticated
listed entry, pinned executable/config/wrapper source, captured public inputs,
and hash-bound local raw receipts. Serving-model metadata may remain null; no
alias, product-quality claim, human acceptance or comparative result follows.
Classic/Government become eligible only after their own native capability and
shared accounting integration are demonstrated with this same approved profile.

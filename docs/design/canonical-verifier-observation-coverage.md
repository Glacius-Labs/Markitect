# Canonical Verifier observation coverage

Status: Host protocol contract for canonical controller verification. The Host's selected canonical scope and evidence remain authoritative for what a Verifier is asked to assess; observation coverage makes a passing response traceable to every declared item in that request.

For a fresh controller verification, Host derives the required observation subjects from the exact selected canonical ScopeIDs, PolicyIDs, every artifact path supplied to the Verifier (including materialized targets and fixed-check input files), and fixed-check identity tuples (`id`, `version`, `digest`). The Host excludes the synthetic Verifier receipt check: that check is computed from invocation provenance, not an obligation the Verifier can assess. Subjects are typed by kind to prevent a path or canonical identifier from aliasing another kind. Fixed-check subjects carry a deterministic opaque token and expose their exact tuple in context so the Verifier can report the expected subject without guessing its encoding.

The complete aggregate must fit `agentexec.MaxVerifierObservations`, which exposes the runner's existing 128-observation response bound. Host refuses an oversized required set before invoking a Verifier; it never truncates obligations or submits a request that cannot be represented in one response.

A `passed` response must contain exactly one observation for every required subject; each must have outcome `passed`. Missing, duplicated, unexpected, blank, or negative observation subjects prevent a PASS. A failed, incomplete, or escalated response may retain a partial set of observations to explain the result, but partial observations cannot be upgraded to PASS. The Host preserves the response outcome and observation outcomes through the existing verification result composition.

This contract validates structural traceability only. A complete set of passing observations is still machine evidence from the configured Verifier. It does not prove semantic correctness, model/provider identity, independence in an organizational sense, authentication, or human acceptance. It adds no Core vocabulary, Module rule, arbitrary obligation language, or capability to waive a fixed check.

This is a forward protocol contract. It does not alter the bytes or identifiers of historical VerificationResults; a verifier-protocol version change applies only to newly created results.

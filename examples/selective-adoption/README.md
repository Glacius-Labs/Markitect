# Selective adoption handoff

This fixture exercises the explicit boundary between selected source evidence and a proposed reusable rule. It is a small, synthetic Git repository created by the end-to-end test; the test builds and runs the real `markitect` CLI. No internal Go records are assembled by the fixture, and no provider or account is involved.

The flow is:

1. `prepare --scope ... --output ...` previews a handoff from exact Git paths at a fixed commit. The preview does not create the destination.
2. `prepare ... --write --expect <handoff digest>` stores the exact selected evidence in a new external directory.
3. `copy-me --workspace ... --queue ... --decision ...` validates supplied support, counterexample, conflict, coverage, uncertainty, and decision records. A successful report still says `adopted: false`.

Run the full CLI proof from the repository root:

```powershell
go test ./examples -run '^TestSelectiveAdoptionCLI$' -count=1
```

The test covers preview without external writes, unchanged source state, exclusion of unselected/private fixture content, handoff creation, validated-but-unauthenticated review, stale expected digest refusal after selected bytes change, and a new capture after an unselected-only commit whose selected snapshot digest remains stable.

## Copy Me record examples

The records below show the complete shapes accepted by the current source validator. Every `$...` value is a placeholder: replace it with the exact repository ID, selected path, file digest, coverage record, or raw-byte hash from your own handoff and report. Do not copy synthetic IDs or hashes as evidence. Keep the candidate in a separate file at `dossier/candidates/rule.yaml`; the queue's candidate path is relative to the directory containing `queue.yaml`.

`candidates/rule.yaml`:

```yaml
apiVersion: markitect.example.org/copy-me-candidate/v1alpha1
stableID: candidate-module-boundaries
proposedRule: "Modules should depend on shared contracts rather than on unrelated modules."
scope: "$EXACT_CANDIDATE_SCOPE"
conditions:
  - "$WHEN_THIS_PROPOSAL_APPLIES"
classification: unclear
support:
  - evidence-support
counterexamples:
  - evidence-counterexample
qualifies:
  - evidence-qualification
confidence: low
confidenceBasis: "$WHY_THIS_CONFIDENCE_FOLLOWS_FROM_THE_SELECTED_EVIDENCE"
alternatives:
  - "$PLAUSIBLE_ALTERNATIVE_INTERPRETATION"
uncertainty:
  - "$WHAT_THE_SELECTED_FILES_DO_NOT_ESTABLISH"
questions:
  - "$OPEN_REVIEW_QUESTION"
```

`queue.yaml` preserves the handoff's coverage record exactly. Evidence paths must identify selected files; each `sourceDigest` is the selected file's `digest` in `handoff.yaml`, not the aggregate snapshot digest. Excerpts are optional and must be exact substrings of selected bytes; omit them when privacy settings do not allow excerpts. Conflict links are explicit and reciprocal in this example.

```yaml
apiVersion: markitect.example.org/copy-me-queue/v1alpha1
evidence:
  - id: evidence-support
    repository: "$HANDOFF_REPOSITORY_ID"
    path: "$EXACT_SELECTED_SUPPORT_PATH"
    sourceDigest: "$EXACT_SELECTED_FILE_DIGEST"
    stance: supports
    observation: "$OBSERVATION_GROUNDED_IN_THIS_SELECTED_FILE"
    conflicts:
      - evidence-counterexample
  - id: evidence-counterexample
    repository: "$HANDOFF_REPOSITORY_ID"
    path: "$EXACT_SELECTED_COUNTEREXAMPLE_PATH"
    sourceDigest: "$EXACT_SELECTED_FILE_DIGEST"
    stance: counterexample
    observation: "$COUNTEREXAMPLE_GROUNDED_IN_THIS_SELECTED_FILE"
    conflicts:
      - evidence-support
  - id: evidence-qualification
    repository: "$HANDOFF_REPOSITORY_ID"
    path: "$EXACT_SELECTED_QUALIFICATION_PATH"
    sourceDigest: "$EXACT_SELECTED_FILE_DIGEST"
    stance: qualifies
    observation: "$LIMIT_OR_CONDITION_SUPPORTED_BY_THIS_SELECTED_FILE"
candidates:
  - stableID: candidate-module-boundaries
    path: candidates/rule.yaml
    digest: "$RAW_SHA256_OF_EXACT_CANDIDATE_FILE_BYTES"
coverage:
  - id: "$HANDOFF_COVERAGE_ID"
    repository: "$HANDOFF_REPOSITORY_ID"
    question: "$EXACT_HANDOFF_QUESTION"
    state: "$EXACT_HANDOFF_COVERAGE_STATE"
    reason: "$EXACT_HANDOFF_COVERAGE_REASON"
requests:
  - repository: "$HANDOFF_REPOSITORY_ID"
    path: "$UNSELECTED_EXACT_PATH"
    reason: "$WHY_MORE_EVIDENCE_WAS_REQUESTED"
    insufficiency: "This path is not selected and was not read. A new owner-approved scope is required."
```

Coverage IDs, repositories and questions must be copied from the handoff; this example also preserves the original state/reason. Later state/reason claims may advance without changing those owner-selected questions. The report keeps both preparation and current coverage. Requests document missing evidence but grant no access. Before writing the queue, hash the candidate's exact saved bytes. For example, in PowerShell:

```powershell
(Get-FileHash -LiteralPath .\dossier\candidates\rule.yaml -Algorithm SHA256).Hash.ToLowerInvariant()
```

Run byte-reference validation once without a decision. The report supplies the candidate, queue, handoff-byte, and handoff-identity digests needed for a separate reviewer record:

```powershell
markitect copy-me --workspace C:/review/capture-1 --queue C:/review/dossier/queue.yaml > C:/review/dossier/report.yaml
(Get-FileHash -LiteralPath C:/review/dossier/queue.yaml -Algorithm SHA256).Hash.ToLowerInvariant()
(Get-FileHash -LiteralPath C:/review/capture-1/handoff.yaml -Algorithm SHA256).Hash.ToLowerInvariant()
```

`decision.yaml` binds one candidate and the exact queue and handoff bytes. Copy `candidateDigest` from the matching `report.candidates[].digest`, `queueDigest` from the raw queue file, `handoffDigest` from `report.handoffByteDigest`, and `handoffIdentity` from `report.handoffIdentity`.

```yaml
apiVersion: markitect.example.org/copy-me-decision/v1alpha1
id: review-decision-001
reviewer: "$SUPPLIED_REVIEWER_LABEL_NOT_AUTHENTICATED_BY_MARKITECT"
date: "2026-10-04"
rationale: "$HUMAN_REVIEW_RATIONALE"
scope: "$EXACT_CANDIDATE_SCOPE"
status: defer
candidateID: candidate-module-boundaries
candidateDigest: "$REPORT_CANDIDATE_DIGEST"
queueDigest: "$RAW_SHA256_OF_EXACT_QUEUE_FILE_BYTES"
handoffDigest: "$REPORT_HANDOFF_BYTE_DIGEST"
handoffIdentity: "$REPORT_HANDOFF_IDENTITY_DIGEST"
```

Then validate the decision against the unchanged inputs:

```powershell
markitect copy-me --workspace C:/review/capture-1 --queue C:/review/dossier/queue.yaml --decision C:/review/dossier/decision.yaml
```

The decision is a supplied, unauthenticated review record. Even `status: accept` would only record that disposition; the report remains `adopted: false`. Changed candidate, queue, or handoff bytes require a new validation and a new decision.

## Public report replay

`replay.go` accepts an already built Markitect executable and a local Markitect checkout. It supplies one fixed revision and exactly one public path, `docs/validation/parallel-wave-konfyra.md`; it does not inspect the checkout itself. The replay is optional because a normal clone or CI checkout may not contain the fixed commit. It creates a temporary external handoff, validates a clearly synthetic Copy Me candidate/decision bound to those exact bytes, then removes the temporary workspace on exit.

```powershell
go run ./examples/selective-adoption/replay.go --markitect C:\tools\markitect.exe --public-repo C:\src\Markitect
```

The replay leaves the source checkout unchanged; it writes the handoff and interpretation records only under a temporary directory and removes that directory when it exits. It prints the handoff identity and the single selected path, but no captured report text. The fixed commit is `93181bb9bc1af0e663b3daa1a4ff772822320307`. The report is public and sanitized; this is evidence only for the selected file and does not imply inspection of the rest of that repository or validation of its claims.

## Limits

This proves explicit selection, fixed-revision byte capture, integrity checks, and separation of interpretation from adoption. It does not prove that the selected evidence is representative, that a proposed rule is correct, that the reviewer is authenticated, or that a validated decision changes Markitect policy. Coverage records describe only the supplied questions. Hashes bind bytes; they do not establish truth, authority, privacy at a remote service, or completeness.

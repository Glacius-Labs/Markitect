# C4 public operating-model proof result

Status: **failed verification; no owner acceptance claimed.** This report records the bounded C4 trial only. It does not change canonical intent, projection policies, check programs, product source, registry, or Core. The fixture contains no adopter/private data.

## Frozen source and fixture

- Product source: `6a79b0d015801f3f37e42cacd049179cc795be42`; the frozen CLI is the v5 binary (`03c551536335ee580b870e703a6a40ef8777c5a8139042f8f0f7e8e7e88729aa`) with receipt digest `6425f113b858f215ecc471a8075e8800f5422c581efb4e3f1dddcda5295c5a53`.
- Driver: `experiments/operating-model-proof/run.py`, SHA-256 `a2dc55d213db619349885b8fd44a85078ce0190402e8f36fe46eb3a6a30d585b`. Provider wrapper: `internal/tooling/codexrunner/runner.py`, SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`. These are different files and both bindings were verified.
- Provider configuration: Codex CLI 0.130.0, model `gpt-5.5`, reasoning `high`, Executor and Verifier. Runtime SHA-256 `af2c31b099c2004ce3ed4792399c910eb24094fab077fea4fc5b2ae32a64cad2`; `record-store` and `private-logs` were under a fresh external path.
- Fixture source origin SHA: `140ffdb9547f025ec2a6e0dd0d2e38f9d0a090c7`. The independent LF snapshot has exactly 29 tracked `examples/operating-model/**` files plus `.gitattributes`; subset commit `b24fd775b3537302ba2ebece970c55ffdef5f7eb`, tree `42b7ffa1e7c219ebe906b14406dc2b2d2018d853`. Source blob/mode manifests matched. The fixture was kept on disposable non-protected branch `codex/operating-model-c4-v5` at the same subset commit.
- Frozen scopes: Orders (`src/Orders/`), Billing (`src/Billing/`), and Product Markdown (`docs/represented/`) with ProductComposition and the two child Definitions. Four declared checks used eight explicit input files.

## Preserved setup and failure attempts

- Public remote lookup of the unpublished fixture-origin commit failed; `c4-source-lookup-01/lookup.json` records that invalid lookup and its resolution through the local source object database. The fixture was exported only from the exact authorized commit.
- V4 setup attempt 01 failed driver preflight because the runtime referenced the wrong repository adapter path. V4 setup attempt 02 failed closed because an empty record-store directory had been pre-created. Both attempts remain in their original run folders. V4 proposal setup attempt 03 succeeded read-only with zero provider calls, writes, or checks; it was not used as v5 evidence.
- Provider calls remained paused until protocol v5 corrected the Verifier reference contract. The v5 driver, wrapper, binary, receipt, and runtime were separately rebound.
- `c4-execute-v5-01` is a driver setup failure, not a protocol event: `execute` was first invoked under a new run ID before a successful proposal binding existed. The driver stopped before controller/native execution. The separate operator note records zero provider calls and zero fixture writes.
- Trial v5-01 then completed Executor and the three outputs were policy-reviewed. Its first explicit write Apply (`9dbacb893e004007a6d537b8058ff44c`) was refused on the protected `main` fixture branch before file writes. However, the attempt initialized a Host-created empty ledger manifest. The second Apply (`670b0c7c6a8144cfa22b1b4652f531b6`) was refused as stale because the prior proposal bound an absent ledger while the manifest now existed. Neither attempt wrote target files or records. The zero-event store was inspected and preserved; no stale candidate was reused.
- These Apply attempts expose a product behavior: a refused write can initialize the ledger before refusing the protected branch, changing subsequent ledger selection. The new trial below binds that actual zero-event state instead of bypassing it.

## Fresh C4 v5-02 lifecycle

The fresh proposal (`c4-module-proposal-v5-02`) bound ledger head `sha256:f77a624521ea48ebd4a7e9f13e5c57619bdee5d32ce7a66fdd4e655d64a8092e` and selection digest `sha256:bbd0dafc21986b54a40a68b028f6166eb5ac6ba8916faf8e7b94125a090ed2d8`. It selected exactly three `work` proposals with input digest `sha256:5193a22361597226aab6e063e25af7b493aac96e492b1955d5de5c3866693573` and proposal digest `sha256:4862af3e3336990012a73bb51b91b65792552978108be03227860af2c0bd1bdf`.

The real Executor completed with status `planned` and run digest `sha256:7763d3ce6bedbbbd1c415aca68c3749b0b1f2aa88206e715e3a14c360eba6a8e`. Billing and Orders each have a provider receipt; Product Markdown was a deterministic projection output without an Executor receipt. The exact output cohort was:

- `src/Billing/BillingQuery.cs`: 683 bytes, SHA-256 `2d5c1df498736970795c64a30719319cfc6a700750518c5d37a99a577ca7cd41`.
- `src/Orders/OrderRules.cs`: 475 bytes, SHA-256 `553264bd9c7249d3e4ac755f0651b10006f32bed7ed8b9a39b0a964f44be7a30`.
- `docs/represented/index.md`: 3,614 bytes, SHA-256 `b1f2e9dd8c4243bddf2eeab0ddf05c265a7b2270cde5e812ade67f6d0b7c68f1`.

The C# outputs matched their exact policies. The Markdown preserved the stated billing/order semantics, ProductComposition relationship, and separate child Definitions. The C4 failure is an adopter-alignment mismatch: the frozen Documentation check asserted for `SumOutstandingCents` and `TryCalculateTotalCents`, while the selected Markdown projection policies did not select the canonical Orders and Billing .NET ProjectionPolicy Definitions as source subjects. The fresh Verifier's structured Product result identified the Markdown policy semantics as present and the overall Product scope as failed because the Documentation check failed. Thus this run does not establish a rendering defect or a candidate-policy failure; it shows the supplied check expects API intent outside the selected Markdown projection scope. The exact reviewed cohort was applied without edits using `--write`.

Apply completed as `materialized-unverified`, attempt `cee3b56086d94db3ab9fb8c2397ab412`. It wrote only the three paths above and produced EvidenceRevision `bf2913533b8042737ce5e3085e5c7aa7c6d03d48`, a commit object whose parent is the frozen subset commit and whose tree is `541e17f99f6001fd206b22b1bdeba98fbcf31abf`. The fixture branch ref remained at the subset commit; no source or canonical files changed.

The fresh independent Verifier ran against that exact EvidenceRevision and produced overall **failed** status (verify attempt `620cbff4fc994d9d84535fb414b5a44c`, output SHA-256 `41bfbc0d67385b1ce296235528a34a4ef0b87413048cfacb2c21f062bbcb24a7`). The four fixed results were:

- `operating-model-orders`: passed, result check digest `811e54a96657af38f30a755da9bd2ddf554f5a01408fe35b3703fbe8d436e60c`.
- `operating-model-billing`: passed, `b58f109e8e1d647ac5af8d782d1a8f72e3c10823c647c0787e271dc848e4d91e`.
- `operating-model-composition`: passed, `0a4d2644b48932c5397e7a6ebb51c25ed94439d6544be995277b7852eda9e50d`.
- `operating-model-documentation`: failed, `55b99ef151625cd3db116ce1d5c4037626af20f5844abd56a1baecf5a62e41b3`; the two missing terms are the method identifiers above.

Billing and Orders Verifier scopes passed. The Product Verifier scope failed because the frozen Documentation check did not align with its selected canonical Markdown projection sources; its structured observations marked the three source Definitions and the ProductComposition relationship passed, and likewise identified the Billing, Orders, and Product Markdown policy semantics as present. The sanitized observation detail hashes are recorded in the event; the private response capture remains external and was not copied here. Verifier input/reference digests are bound in the structured event (Billing 5 refs, Orders 5 refs, Product 17 refs). This classification comes from the actual structured Verifier response and fixed-check result, not the earlier artifact-only inference about rendering.

## Evidence and limits

All v4 setup failures, the v5 wrong-run-ID driver error, both v5-01 Apply failures, both C4 control freezes, the exact proposal/Executor/Apply/Verify events, and this report are retained under the C4 run paths. Provider logs and raw structured output remain in the external temporary trial root; this repository contains only sanitized structured metrics, event hashes, candidate hashes, and this summary. No unknown path is claimed adopted. No C5/C8 trial, candidate repair, canonical/policy/check change, or owner acceptance claim is included.

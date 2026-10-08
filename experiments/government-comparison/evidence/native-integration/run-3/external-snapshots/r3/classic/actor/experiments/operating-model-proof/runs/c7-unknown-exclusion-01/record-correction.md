# C7 result record correction

`controller-propose-result.json` is retained as committed historical evidence but is classified `INVALID_RECORD_FORMAT` (strict JSON rejects its trailing comma) and `INVALID_EVIDENCE_MAPPING` (it presents operator-measured content hashes as though they were fields in the CLI inventory/exclusion response). Do not use that file as the product observation.

`controller-propose-result-v2.json` is the corrected record. Its product-observation section is extracted directly from the preserved 10,219-byte CLI stdout (SHA-256 `2dd3a86e794fec27fddbc04a8f121424bc24bcc9879079af70a043af2fe04440`) and validated with Python's strict JSON parser. It preserves the actual inventory entries as path/mode/size and the exclusion as path/reason/state/metadata path/mode/size; it does not attach hashes absent from the CLI output. Fixture content hashes are retained only in the clearly labelled operator-measurement section, sourced from the pre-call freeze and a post-call read-only snapshot.

The CLI was invoked once and returned `escalated` with `artifact.unowned`; no executor, verifier, agent process, provider, Execute, Apply, or Verify was invoked. The fixture and copied ledger were not mutated.

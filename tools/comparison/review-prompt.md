# Reviewer prompts

Frozen prompts for the artifact review. Each review is one fresh reviewer session with read access to exactly one token directory, the requirements file and the rubric. `TOKEN_DIR`, `REQUIREMENTS` and `RUBRIC` are replaced by absolute paths; `FINDING` by one finding's JSON. Nothing else is added.

## Review

```text
Review the program in TOKEN_DIR against the requirements in REQUIREMENTS, following the rubric in RUBRIC exactly. TOKEN_DIR contains candidate/, start/ and diff.patch. Read the requirements and the rubric first, then the candidate's documentation, tests and the diff; read production code where an item needs it. Do not run the program or its tests. Do not read anything outside these three paths. Answer with the single JSON object the rubric specifies, with "token" set to the name of TOKEN_DIR.
```

## Confirmation of a critical finding

```text
A reviewer reported the finding below about the program in TOKEN_DIR, judged against the requirements in REQUIREMENTS and the rubric in RUBRIC. Decide independently whether it is correct: the cited text exists, and it really contradicts the quoted requirement (or, for R4 and R5, really is the reachable or unjustified thing claimed). Omission is not a contradiction, and a different correct design is not a finding. Do not read anything outside these three paths. Answer with one JSON object: {"confirmed": true or false, "reason": "one or two sentences"}.

FINDING
```

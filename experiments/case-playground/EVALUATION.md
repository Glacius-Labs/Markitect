# Evaluation layer and case series v2: blueprint

Stage 2 of the playground. It adds what the first build lacks for judging the
method: predeclared ground truth, hidden holdout checks, an independent consistency
review per wave by two model providers, a realistic longer change series, failure
classification, and provider-independent outer agents. [BLUEPRINT.md](BLUEPRINT.md)
stays the base design; this file extends it and wins where they differ.

It follows the product's evaluation guidance on branch
`claude/vision-concept-record-20261010`: `docs/measurement.md#evaluating-the-method-separately-from-its-runtime`,
`docs/concepts/register.md` (DEC-008, DEC-009, DEC-010, ENH-004, ENH-005, OQ-005).

## Principles

1. **The product supplies the method.** The harness only selects and configures what
   Markitect offers (setup, MCP, its own inner roles). No harness-side Manager,
   review or verify logic, no harness executor for Markitect's inner roles.
2. **Classify every failure.** `product` (Markitect setup, runtime, MCP or tool
   failure; recorded with cause and handed to the product side), `harness` (our code),
   `environment` (Docker, network, provider/auth/rate limit). None of them is an
   outcome of the method.
3. **Ground truth is fixed before any run** and never visible to an arm. Holdout
   checks test only what the public requirements and project rules already say:
   hidden cases, never secret requirements.
4. **Assessment is separate from execution.** It runs after the run, in its own
   container, on the frozen station snapshots. Nothing from it can reach the agent.
5. **Two providers review.** Every wave is reviewed independently by a Codex reviewer
   and a Claude reviewer, both different models from the arms. Disagreement is data.
6. **Provider independence.** The outer agent can be Codex or Claude Code. A run's
   provider is a recorded factor; runs with different outer providers are never
   pooled as a method effect.
7. **Cost is secondary.** Primary scores are missed and unnecessary inclusions, rule
   violations, contradictions, regressions and escalations.

## Case series v2: `readinglog2`

A long-lived brownfield project with six waves. Keep `readinglog` and `roombook`
unchanged (v1, for the shakedown pair and smoke tests). `readinglog2` reuses the
Readinglog domain but its baseline already establishes three cross-cutting project
rules that later backlog items do not repeat. That is where agents tend to forget them.

### Project rules (in the public README from the start, implemented by the baseline for add/list)

- **R1 Audit log.** Every successful mutating command appends exactly one JSON line to
  `<db>.log`: `{"op": COMMAND, "ids": [IDS in processing order], "at": UTC ISO-8601}`.
  Failed commands and read-only commands append nothing.
- **R2 Error contract.** Every domain/input error prints
  `{"error": {"code": CODE, "message": TEXT}}` and exits 2 without any mutation. Codes
  come from the README's error-code table (baseline: `invalid_input`, `duplicate`,
  `not_found`, `storage`). A new code must be added to that table.
- **R3 Text normalization.** All text input, from arguments or files, is Unicode NFC
  normalized, stripped, and internal whitespace runs collapse to one space. IDs are
  case-sensitive after normalization.

### Waves (6 waves, 15 items)

| Wave | Items | Character |
|---|---|---|
| S1 | B01 `finish --id` (idempotent) and `list --status` | small feature, touches R1/R2 |
| S2 | B02 atomic `import --csv`; B03 `summary` per author; B04 integrate tests/docs | features; import must apply R1-R3 to file input |
| S3 | B05 `list --author`; B06 `export --csv`; B07 `summary --author`; B08 tags (`tag --id --add/--remove`, `list --tag`, records gain `tags`, export gains a `tags` column joined by `;`, stored records without tags stay readable without rewrite on read); B09 group tests; B10 integrate + TEAMWORK.md | team wave with real cross-group interaction (tags x export x list) |
| S4 | B11 storage refactoring: the file becomes `{"schemaVersion": 2, "records": [...]}`; reads accept v1 arrays and v2; read-only commands never rewrite; the first successful mutation writes v2; CLI output unchanged; B12 tests on v1 fixtures and docs of the formats | refactoring with behavior preservation |
| S5 | B13 late decision change: pages become optional (`add` without `--pages`, empty CSV cell) and are stored as `null`; `summary` sums known pages and adds `unknownPages`; `export` writes an empty cell; positive-integer validation stays for given pages; B14 tests/docs across all affected commands | revisits an S1/S2 decision across many features |
| S6 | B15 rename book -> entry (JSON keys `entries`, summary `entries`), with `--legacy` on list/summary for one compatibility period, composable with all filters including `--tag`; stored data and audit op names unchanged; CSV headers unchanged | late cross-cutting rename |

The backlog text for B13 names the decision, not the list of affected commands.
Finding them is the point. The same holds for B11 and B15.

Public checks (`cases/common/checks/acceptance.py`, case branch `readinglog2`) stay a
small visible set per wave, like v1. They do not cover the rules exhaustively.

### Hidden evaluation files (`evaluation/readinglog2/`, never staged into a run)

- `ground-truth.json`:
  `{"case", "rules": [{"id","text"}], "waves": [{"station", "items": [{"id",
  "obligations": [text], "areas": [text], "rules": [{"id","expectation"}],
  "mustNotChange": [text]}]}]}`. Areas are behavioral and documentary areas
  (commands, README sections, error-code table, storage validation, tests), not file
  names, so they fit any architecture.
- `holdout.py`: `python3 holdout.py --repo DIR --station N [--deadline SECONDS]` prints
  one JSON object `{"station": N, "checks": [{"id", "status": "PASS"|"FAIL"|"ERROR",
  "item", "rule", "detail"}]}` and exits 0 even when checks fail. ERROR means the
  holdout could not judge (for example its deadline passed); the assessment counts it
  apart, never as a failure of the candidate. It runs the CLI on fresh temp data
  like `acceptance.py`. Station N runs every holdout released up to N, so earlier
  waves are re-checked for regressions. Every holdout names the public sentence it
  derives from (item or rule id).
- `reference/`: a hidden reference implementation per wave, used only to validate the
  public checks and holdouts (all pass on the reference; targeted mutants such as
  "import without audit line" or "export ignores tags" must fail the intended
  holdout). Never staged, never shown to reviewers.

## Assessment (`python -m playground assess --run DIR`)

Runs after `host run`, on the host, launching one assessment container
`mpg-assess-<id>` from the same image. Mounts: the run folder read-only, the case's
`evaluation/<case>/` read-only (when present), reviewer credentials read-only, and
`<run>/assessment/` writable. Per station, on a scratch copy of that station's
`immutable-main` (no `.git`) as an unprivileged user:

1. **Holdouts** (when the case has them) and the public checks again.
2. **Diff profile** of the wave: files and lines changed between the previous
   station's main and this one, split into `model` (`.markitect/`), `code`, `tests`,
   `docs`, `config/other`. This is the proxy for human review effort.
3. **Consistency review** by each reviewer (below).
4. **Failure classification** of the station and **product findings**: setup steps,
   MCP tool calls to the `markitect` server that failed, product errors in the agent
   events, with the exact error text, collected into `assessment/product-findings.md`
   for the product side.

`assessment/report.md` and `report.json` aggregate per wave: holdouts passed/total
with failures listed by item and rule, reviewer findings by category for each
reviewer and their agreement, diff profile, escalations, classification.

`python -m playground compare RUN_A RUN_B` writes a side-by-side comparison of two
assessed runs. It first checks the fairness fields (case, image, outer provider,
model, effort, limits, container) and refuses to compare mismatched runs unless
`--allow-mismatch` is given, which is then printed at the top.

### Reviewers

- **Codex reviewer:** `codex exec` with model `gpt-6.1-sol`, effort `high`, sandbox
  `read-only`, `--output-schema`, fresh CODEX_HOME with only `auth.json`.
- **Claude reviewer:** Claude Code `claude -p` with model `claude-opus-5-5`,
  `--output-format json --json-schema`, read-only tools only
  (`--allowedTools Read Grep Glob`), `--bare` or
  `--setting-sources` so no user settings load, fresh `CLAUDE_CONFIG_DIR`.
  Authentication via `CLAUDE_CODE_OAUTH_TOKEN` read from a token file the operator
  creates once with `claude setup-token` (`--claude-token PATH`). Never copy the
  host's `~/.claude` credentials: the operator's own Claude session uses them.
- Reviewer models and efforts come from an evaluation config with these defaults;
  both are recorded in the report.
- **Input per wave:** the released items' public text, the project rules, the wave's
  ground truth (when present), the wave diff (`git diff` between the two mains,
  reconstructed from the bundles), read access to the station snapshot, and the
  agent's final message of that wave (for honesty and escalation). No arm label in the
  prompt; the repository may still reveal the method (`.markitect/`), which the report
  states as a blinding limit.
- **Output schema:** `{"findings": [{"category": "missed_obligation" |
  "unnecessary_change" | "rule_violation" | "contradiction" | "regression" |
  "false_claim" | "escalation_needed" | "escalation_unneeded", "severity":
  "high"|"medium"|"low", "item", "rule", "evidence": [{"path","line"}], "detail"}],
  "obligations": {"covered": int, "total": int}, "notes": str}`. For Markitect,
  contradictions include model vs code/docs/tests.
- One fixed, versioned prompt file `evaluation/common/reviewer-prompt.md`, identical
  for every arm and provider.

## Run-side changes

- **N stations.** Stations come from the case's `STATIONS.json` (v2 has six). The
  runner, lifecycle, report and manifest must not assume four.
- **`agent.kind: claude`.** Claude Code as the outer agent (pinned npm version in the
  image, e.g. `@anthropic-ai/claude-code@2.1.296`):
  `claude -p --output-format stream-json --verbose --model M --effort E
  --dangerously-skip-permissions` in the container, `--resume <session>` from S2 on,
  `--mcp-config` with `--strict-mcp-config` for Markitect's server, fresh
  `CLAUDE_CONFIG_DIR`, token from `--claude-token`. Usage from the stream's result
  events. For both methods, when the outer agent is Claude and the repo has no
  `CLAUDE.md`, the harness adds a one-line `CLAUDE.md` containing `@AGENTS.md`
  (recorded, identical for both arms). Markitect's inner roles stay on Codex (product
  limit, ENH-005), so a Markitect run with a Claude outer agent needs both logins and
  is reported as stratum `outer=claude, inner=codex`.
- **Role configuration.** After Markitect setup, read `.markitect/runtime.yaml` and
  record the actual executor, model and effort per role in the report. Per-role
  manifest fields come later, when the product's setup can configure them (ENH-004).
- **Classification** of the run outcome (`product`/`harness`/`environment`/`none`)
  in `report.json`, with the reason.
- **Image:** add `python-is-python3`; agents expect `python`.

## Pre-registration

Ground truth, holdouts, reviewer prompt and evaluation config are committed before
the first v2 run. `assessment/report.json` records the evaluation commit and file
hashes, so a run is always judged by the evaluation that existed before it.

## Out of scope

Statistics across runs, egress filtering, executors for Markitect's inner roles,
mixing Windows and Linux runs.

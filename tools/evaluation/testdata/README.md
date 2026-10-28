# Recorded test data

Recorded runs and transcripts for the offline tests of the failure classifier and the token parser ([test_classify.py](../test_classify.py)). Message content is removed; structure, IDs, error fields and usage are kept as recorded.

## `claude-transcript-subagents/`

A real Claude Code 2.1 session recorded on 27 October 2026 (a workflow probe on Claude Haiku) with 22 workflow subagents, in Claude Code's own layout: the session's transcript `SESSION.jsonl` and its subagents' transcripts under `SESSION/subagents/workflows/*/agent-*.jsonl`. Lines of types that carry no usage (attachments, API request records) are dropped; every assistant line and the session's `cost-state` are kept.

The hand count, done line by line with no code of the parser's:

```text
grep -rh '"type":"assistant"' . \
  | sed -E 's/.*"id":"([^"]+)".*"input_tokens":([0-9]+),"cache_creation_input_tokens":([0-9]+),"cache_read_input_tokens":([0-9]+),"output_tokens":([0-9]+),"output_tokens_details":\{"thinking_tokens":([0-9]+)\}.*/\1 \2 \3 \4 \5 \6/' \
  | sort -u | awk '{n++; a+=$2; b+=$3; c+=$4; d+=$5; e+=$6} END {print n, a, b, c, d, e}'
```

It gives 27 responses (3 in the main session, 24 in subagents), 56 input, 413,772 cache-creation, 638,602 cache-read, 2,364 output and 1,368 thinking tokens. The session's own `cost-state` states the same totals. The main session's transcript alone holds 3 responses with 38,102 cache-creation tokens: a count over the main transcript only would miss most of the run.

## `failures/`

One directory per recorded run, as `suite.py run` leaves it: `suite.json` (engine steps and the container argv dropped) and what the classifier reads of the variant's output.

| Directory | Source | Expected class |
| --- | --- | --- |
| `infrastructure-not-started` | E1 `probe-01`: Git Bash rewrote the variant's path, so the variant never ran ([E1 record](../../../docs/validation/eval-e1-sandbox-smoke-20261028.md#results)) | void, infrastructure |
| `provider-overloaded` | A recorded Claude Code API error `529 Overloaded` (`error: server_error`, 18 August 2026) as the session's last response, in E1 `k-02`'s record with the exit code K gives a failed session | void, provider |
| `provider-usage-limit` | A recorded subscription limit error (`error: rate_limit`, `apiError: usage_limit_reached`, 10 October 2026) as the session's last response, in the same record | void, provider |
| `provider-recovered` | The recorded 529 error followed by a further response of the same session | result |
| `provider-subagent` | The recorded 529 error as a subagent's last response, while its session went on | result |
| `variant-capture-k01` | E1 `k-01`: K's capture dropped sources; no signature named it before the series | result |
| `result-k03` | E1 `k-03`: an ordinary delivered run | result |

# Actual runner probe r1

This is the explicitly authorized real native start, not another version smoke.
Exactly one of two permitted start attempts was consumed. The requested model
was `gpt-6.1-sol`, reasoning `high`; no fallback was selected. Codex exited 1
after 0.178 seconds with a configuration rejection: built-in provider `openai`
cannot be overridden. There were no JSONL events or started agent turns. Actual
provider model, request count, usage and effective context/access remain unknown.

`request-1.json`, `prompt-1.txt`, `result-1.json`, `session-1/*` and the private-to-
operator probe-budget SQLite ledger are unchanged copies of the external public-
sentinel probe directory. The ledger contains only this public probe's attempt
metadata, no credentials. No study source or private test data was sent to an
Actor. Original absolute receipt paths stay intact; map paths relative to the
observation's `originRoot` into this directory and verify hashes.

`invoked-source/` preserves exactly the three source files digested by the actual
request. After this attempt, current `identity_probe.py` was corrected to close
its SQLite connection explicitly. This was found by a Windows temporary-test
cleanup failure, not by another Actor attempt. The original failed
`targeted-tests.log` remains; `targeted-tests-r2.log` records three passing checks
after the fix. `process-deadline-test.log` records the focused descendant-deadline
check. No general mechanical sweep or full Go/Verify suite was repeated.

The Overseer's addendum removed the six-internal-provider-request cap only for
this tiny probe. Its replacement limits are two attempts, one wrapper agent turn
per attempt, 180 seconds per owned Windows Job, parallel 1, no continuation/retry
or subagents, and a retrospective 10000-token stop threshold when counters arrive.
Agent-turn events are never equated to internal provider requests. The process
wrapper observer supports stopping on reported threshold/unexpected activity;
the actual rejected start produced no usage counters and did not exercise such a
provider-token stop. Time/session/process control remains independent.

Probe 2 was conditional on executable Probe 1 and observed remaining budget.
That condition failed, so no second attempt occurred. The unsupported retry
settings are recorded, not silently renamed or replaced. The later common study
profile and Classic v0.14.1/c913 candidate remain unchanged. S1 stays open.

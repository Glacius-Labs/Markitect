# Study in brief

## Question

For an ordinary developer working with Codex CLI or Claude Code, what does Markitect add, good or bad,
compared with strong conventional agentic coding on the same work? We look at the
delivered behavior, consistency with the stated rules, regressions, how easy the code
is to change and continue, and how much work it took to get there.

## The two arms

- **Conventional:** clear project rules (AGENTS.md plus a short working-method
  fragment), planning, tests, native subagents, review and self-directed repair. It is
  not weakened and does not have to imitate Markitect.
- **Markitect:** the real product installation as a user would set it up (project
  init, onboarding, runtime setup, MCP server) and its normal workflow, including the
  inner roles it schedules. Their cost counts. The harness does not author the model;
  the agent does that as part of its work.

## Fairness

Both arms use the same container image, outer agent (Codex or Claude Code, its
version, model and effort), prompt (identical for every wave), case files and time
limits. Runs with different outer agents are never pooled; a Markitect run behind
Claude Code still runs the product's inner roles on Codex (stratum
`outer=claude, inner=codex`). The subagent limit is a Codex
setting per session and is set for the outer agent only; Markitect's inner roles are
separate sessions whose limits the product sets, so total concurrency is not equal by
construction. Time limits count agent time; checks and snapshots between waves are not
charged. Each run starts from a fresh repository and a fresh agent context, then
resumes the same session across the waves (a resume that starts a new session is
flagged in the report). Nobody coaches the agent between waves; the harness only
releases the next wave. Check results, old runs and the other arm's results are never
inputs. Product failures that a normal user would hit are findings, not something to
work around.

Run the two arms of a pair one after the other (they share one Codex login), alternate
which arm goes first from pair to pair, and keep both run folders: `host.json` records
when each ran. Before comparing, check that the "Fairness" lines of both reports match.

## What is measured

| Evidence | Meaning |
|---|---|
| Public checks per wave and at the end; the candidate's own tests | Outcome quality, the same for both arms |
| `markitect project check` on the final state | Markitect conformance, reported separately; a valid model alone is not correct software |
| Setup time and every setup command | Cost of installing the method |
| Wall time, agent exit, timeouts, commands, MCP and subagent calls, commits on main | Effort per wave; unknown values stay unknown, never zero |
| Tokens per wave: the outer session, and all sessions Codex recorded (subagents, Markitect's inner roles) | Cost; ephemeral sessions keep no record, so "all" is a lower bound |
| Leftover processes, infrastructure errors, stop reasons; each failure classified as harness, environment or product | Validity of the run; product failures go to the product side |
| Hidden holdouts per wave (cases with predeclared ground truth) | Missed obligations and regressions the public checks do not cover |
| One review per wave by a Codex and a Claude reviewer, and their agreement | Missed and unnecessary work, rule violations, contradictions, false claims, escalations |
| Diff profile per wave (model, code, tests, docs, other) | Proxy for human review effort |

The assessment ([EVALUATION.md](EVALUATION.md)) uses the same rubric, prompt and
holdouts for both arms and adds cases only from the public requirements. Reviews are
evidence, not human acceptance.

## What one run can and cannot show

One run per arm shows concrete mechanisms: where an arm failed, what it cost, what the
method changed in the work. It cannot show a general effect, a winner, statistical
confidence or an economic benefit. The waves of one project are dependent
observations, not independent samples. These small synthetic projects approximate a
bounded user workflow; they say nothing certain about real repositories or human
acceptance. Keep observed facts, likely explanations and untested ideas apart.

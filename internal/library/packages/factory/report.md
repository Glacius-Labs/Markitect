# Report

The report of one session or run of an agent: what it did and what struck it. Every agent writes one per session or run into its folder `.markitect/context/AGENT/`, named like its Agent document, named by date, time and subject, such as `2026-11-24-1432-renew-a-loan.md`; an agent without a subject, such as an observer or the Triager, leaves the subject out. A run of an observer that observed nothing writes no report, only its checkpoint, so the context holds no "all well". A report is information: it demands nothing and stays unchanged. Take it for every session and run. Do not take it for a finding: the Triager makes findings from reports.

## Started by
Who started the session or run: a person, `Norman`, `Housekeeper` or `routine`; for an agent that another agent started, such as a Reviewer, that agent. Example: `Housekeeper`.

## For
The person behind it, as a link, or `the project` for a run nobody asked for. Example: `[Ada](../../../docs/spec/factory/people/ada.md)`.

## Started
When it started: `YYYY-MM-DD HH:MM`. Example: `2026-11-24 14:32`.

## Finished
When it ended: `YYYY-MM-DD HH:MM`. Example: `2026-11-24 15:05`.

## Outcome
How it ended, in a word or two, such as `merged`, `aborted`, `planned`, `accepted`, `observed` or `source unreachable`. An aborted session states its reason under Statements. Example: `merged`.

## Item (optional)
The plan item the session worked on, as a link. Example: `[Renew a loan](../../plan/lending/renew-online/renew-a-loan.md)`.

## Statements (optional)
What the session or run observed or wants known, bundled by theme under level-3 headings and numbered throughout, so that a finding can point at statement 4. Each statement is one sentence, with numbers, and the reference to its place; for a source outside the repository the excerpt in the source's wording. Example: `3. Renewing failed twice for loans older than a year. [run 412, renew.test.ts]`.

## Not observed (optional)
What the run could not see this time, such as a source without access. Example: `The test logs of run 412: not downloadable.`

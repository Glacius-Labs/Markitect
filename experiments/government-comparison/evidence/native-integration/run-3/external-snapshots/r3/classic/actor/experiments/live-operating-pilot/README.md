# Live operating pilot

This is the durable, bounded input package for one synthetic commerce change across .NET and Markdown projections. It freezes the canonical fixture at baseline 9dc9e68c182bccd408690f9f20300a75c68bd35a and changed intent bd06f90c9afcc80f53f5c70d355b39266b6c4331; the only canonical difference is the UseCase quantity bound changing from 1..20 to 1..10. The fixture manifest records every tracked source-file digest and the selected module/projection identities.

The source bundle contains only those committed fixture revisions. It excludes uncommitted target outputs. bridge.py, protocol-final.json, and holdout/harness/run_acceptance.py are frozen public experiment tooling. The bridge transports actual externally supplied collaboration-agent responses and preserves a declared changed-source .NET Executor negative control; it does not call a model or synthesize a verifier finding. Fresh independent collaboration agents and the actual Host invocation are supplied by the external runner.

## Prepare an exact fixture

Requirements: Git and Python 3.9 or later. Choose a new, absolute destination outside the Markitect checkout; the script refuses existing destinations and never deletes or overwrites paths.

    python experiments/live-operating-pilot/prepare.py --destination C:/tmp/live-operating-baseline --revision baseline
    python experiments/live-operating-pilot/prepare.py --destination C:/tmp/live-operating-changed --revision changed

The script clones the local source bundle without checkout, sets repository-local core.autocrlf=false, and then checks out the exact frozen commit on a named non-protected feature branch. It performs no network access. Use the baseline fixture for baseline proposal/implementation. To continue the same lifecycle, retain its generated targets and external ledger, then switch that same fixture to a new feature branch at the changed commit. Preparing a separate changed fixture is useful for inspection but does not reproduce that retained-state lifecycle.

## Acceptance harness

holdout/harness/run_acceptance.py is a deterministic offline .NET behavior check for Commerce.CreateOrderHandler. It needs the installed .NET SDK 10.0.103. It checks rejected/accepted quantities, callback count and unchanged callback argument across multiple instances and call orders; it verifies that the candidate source stays unchanged. Use the maximum matching the frozen revision (20 for baseline, 10 for changed), for example:

    python experiments/live-operating-pilot/holdout/harness/run_acceptance.py C:/tmp/live-operating-changed 10

The harness writes source copies and build/execution logs under its sibling holdout/runs/, and a result JSON under holdout/results/. Run it from a disposable external copy of this package so generated runtime evidence stays outside the tracked source package.

## Bounds and evidence status

This is one narrow synthetic slice with two related projections. The fixed protocol bounds each bridge wait to 540 seconds, each host runner to 600 seconds, and semantic repair to one attempt. The negative control applies only to the changed-source .NET Executor response without repair; verifier responses remain untouched. The protocol records the planned stages and limits, including that this is not a statistical reliability or productivity study.

The [sanitized result receipt](result.json) records 15 actual Luna High actor invocations across three explicitly versioned attempts. Baseline behavior passed 120 calls; the controlled fault failed 10 checks across 80 calls; the repaired state passed 80 calls, fresh leaf and parent verification, and a complete controller audit. The first two attempts stopped on an omitted owned project file and malformed role envelopes respectively. Their results and continuation protocols remain alongside the final result. This is iterative engineering evidence, not a first-pass success. Only the handler changed during the final repair; the project file and Markdown stayed byte-identical. The [validation report](../../docs/validation/standard-operating-model.md) explains source changes, evidence boundaries and remaining work. Do not add raw actor prompts, private responses, queue contents, private logs, or machine-specific temporary paths to this package. A successful compile check alone does not establish behavioral acceptance; the independent verifier and holdout results, bounded repair, parent composition, and controller audit must be reported as separate evidence.
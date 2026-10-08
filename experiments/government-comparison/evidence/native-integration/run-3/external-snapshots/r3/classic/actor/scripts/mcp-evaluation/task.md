# MCP evaluation runbook

The delivery-service fixture owns the single shared actor prompt at [`query-only-prompt.md`](../../examples/onboarding/delivery-service/tasks/query-only-prompt.md) and the evaluator-only oracle at [`query-only-expected.yaml`](../../examples/onboarding/delivery-service/tasks/query-only-expected.yaml). Keep task wording and expected answers there; do not duplicate them in this runbook.

The recorded baseline is `973f11d9cfc157610ed84f4e9582c3534c67018c`; the published CLI reports snapshot digest `3bb9fe30d2188c86bcce1bf77fc551bcbd6741f4728ea34f270f7d9cc3668e87`. The separate CLI-only mutation candidate is `26c88c133cf09e09dd7f03a630c8e470128c002d`.

Use `run-transport.ps1` to call published CLI `find`, `explain`, and `context` against the pinned baseline, run the local MCP stdio probe, compare exact returned text payloads, and exercise protocol boundaries. It is a deterministic transport check only. It is not a substitute for a real authoring-client run, model comparison, or productivity evidence.

The prototype fixes repository and revision at server startup. Its tool schemas reject per-call `repo`, `revision`, arbitrary `path`, and `package` properties. Report those rejections as schema boundaries, not package-aware functionality. Preserve any authoring-client startup or tool error as an unscored run; record model tokens only when the client reports them.

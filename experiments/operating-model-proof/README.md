# Operating-model proof driver

Protocol and claims are frozen in [protocol.md](protocol.md). This directory holds the driver and later sanitized, immutable run events under runs/<run-id>/. Historical protocol-v1 evidence under experiments/capability-proof/ is not modified.

The driver is a staged wrapper around the real controller CLI. It never emits agent candidates itself. Run each stage separately:

1. Create a fresh private Git clone of examples/operating-model and commit the exact source fixture. Keep the clone and all target roots separate from this Markitect checkout. Prepare an external runtime JSON with fresh absolute RecordStore and privateLogs directories and the exact Executor/Verifier bindings in protocol.md.
2. Call propose and then execute with the same run ID, fixture, full base/candidate SHAs, CLI executable, config path and runtime. Execute requires the fixture HEAD to equal the candidate revision and the source tree to be clean.
3. Review the external reviewed-run JSON and its digest. Call apply with --write, --reviewed-run and the exact --expect digest. Do not infer owner approval from this review step or from technical output.
4. Call verify with --base set to the canonical source revision and --revision set to the immutable evidence revision reported by apply. Use --write to append verification results.

Example shape:

    python run.py propose --run-id c3-positive-01 --fixture C:\proof\fixture-c3 --cli C:\proof\markitect.exe --config canonical.yaml --runtime C:\proof\private\runtime.json --external-root C:\proof\private\staging --base FULL_BASE_SHA --revision FULL_CANDIDATE_SHA

Repeat the same arguments with execute. Then pass the exact external reviewed-run path and digest to apply. After apply, call verify with the source and evidence revision pair.

Each action captures actual stdout/stderr digests and elapsed process time. Execute's complete candidate JSON is staged under external-root/<run-id>/reviewed-run.json for the guarded Apply command. Provider JSONL logs remain at runtime.privateLogs. Repository run events include digests, statuses, scopes, file sizes, actual receipts and available telemetry, never candidate bytes or provider logs. Do not copy external staging or private logs into this directory.

The script fails closed on non-full revisions, dirty pre-apply fixture source, CLI/runtime changes within a run, runtime digest mismatch, missing Codex/model bindings, output over the runtime ceiling, overlapping private paths, or mismatched reviewed-run digest. It requires explicit --write for Apply and Verify.

No real agent trial has been run from this directory. C9 remains BLOCKED for the current fixture arrangement as described in protocol.md.

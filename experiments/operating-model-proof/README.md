# Operating-model proof driver

Protocol v1 preserves historical C5 FAIL and prior NOT RUN statuses. Protocols v2-v4 freeze fixture, source, and earlier runner-contract evidence. Protocol v5 separates Module-authorized new output paths from immutable evidence references, requires complete verifier references, and fixes assurance-count reporting. Read [protocol-v1](protocol.md), [protocol-v2](protocol-v2.md), [protocol-v3](protocol-v3.md), [protocol-v4](protocol-v4.md), and [protocol-v5](protocol-v5.md) before creating a run.

The staged Python driver calls the Markitect controller CLI. It never creates agent output. Run propose, execute, reviewed apply and verify as separate invocations. Use a fresh private clone of the public fixture and unique external RecordStore, private-log and staging paths for each run.

Every invocation requires an external CLI build receipt with full sourceSha, actual buildCommand, integer exitCode 0, and binaryDigest matching the selected CLI bytes. Source SHA must resolve to a full Git commit in the repository. The driver binds its digest and source to the run. Runtime configuration binds Codex CLI 0.130.0, model gpt-5.5, high reasoning, Python 3.13, the repository's codexrunner adapter and the native Windows x64 executable. Raw provider logs, CLI output and candidate bytes stay in external directories; repository events contain digests and bounded observed fields only.

Example shape:

    python run.py propose --run-id c3-positive-01 --fixture C:\proof\fixture-c3 --cli C:\proof\markitect.exe --config canonical.yaml --runtime C:\proof\private\runtime.json --build-receipt C:\proof\private\markitect-build.json --external-root C:\proof\private\staging --base FULL_BASE_SHA --revision FULL_CANDIDATE_SHA

Repeat with execute and the same source/runtime bindings. Review the external reviewed-run JSON and its digest, then call apply with --write, --reviewed-run and the exact --expect digest. After apply, call verify with --base set to the canonical source revision and --revision set to the immutable evidence revision, plus --write.

For protocol-v2 C9 negative controls, first create the run binding with a clean-fixture propose. Use the normal canonical candidate plan/apply CLI actions separately for the frozen incorrect parent bytes; preserve their complete output only in external staging and do not hand-edit their emitted records. Then call this driver’s verify action with the source and object-only evidence revisions. The v2 protocol classifies the wrong parent bytes as controlled non-AI materialization, never as an Executor result.

The driver stores unique raw captures outside the repository. Each retry gets new exclusive stdout/stderr and reviewed-run paths so prior failures remain available. The public event records exact action/flags with local paths replaced by logical placeholders, input/output digests, exits, elapsed time, scopes, record IDs and usage only when returned.

Static driver tests run with:

    python -m unittest discover -s experiments/operating-model-proof -p test_run.py

These tests validate evidence boundaries only; they are not product checks or capability trials. No private real-agent trial is established by this directory until a new uniquely bound run record is executed.

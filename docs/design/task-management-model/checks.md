# Checks

Each Check defines a method and expected observations. None has run against a task application. Relations specify exactly which contracts it examines; passing an implemented subset does not establish all obligations. Record eventual source revision, environment, selected file, procedure, outcome and failures separately from intent status.

<a id="check-domain"></a>
## CHECK-DOMAIN — Task meaning

Type: Check. Method: automated domain/contract tests.

Test normalized text, whitespace-only text and rejection before normalization of tabs, CR/LF, terminal escapes and Unicode U+0085/U+2028/U+2029. Accept ordinary Unicode and duplicate wording. Test reference and state validation, including reference allocation/order beyond a machine-sized integer, invalid states and duplicate references in a decoded collection. This examines valid records, not usability or filesystem safety.

<a id="check-actions"></a>
## CHECK-ACTIONS — Offered actions

Type: Check. Method: automated action tests with controlled storage.

- Add to an empty collection: reference 1 and open state. Add again: reference 2, including identical text. Failure before saving leaves no added task or consumed reference.
- List: empty message or header plus all tasks in numeric reference order, including completed tasks. Listing does not write.
- Complete: transition an open task; retain text/reference and other tasks. Repeat completion: successful no-op without a write. Unknown valid reference: refusal without a write.

Observe the UseCase results and storage calls independently. This does not prove the real CLI or persistence adapter works.

<a id="check-cli"></a>
## CHECK-CLI — Command-line interface

Type: Check. Method: invoke the built executable as a real child process.

Exercise every command and its expected stdout/stderr/exit code, quoted text, unknown command, missing/extra arguments, invalid reference syntax and invalid text, including the forbidden control/separator characters. Verify the default file and explicit absolute/relative file selection, including syntax failures for repeated options and empty paths. Confirm stdout remains empty on failures and stderr on success. Normalize only newline conventions, not wording, tabs or ordering.

Use a temporary working directory and known selected files. This proves observed interface cases, not every shell's quoting behavior or human understandability.

<a id="check-restart"></a>
## CHECK-RESTART — Durable complete interaction

Type: Check. Method: integration test using separate process invocations and the real storage adapter.

Start with a missing file. List without creating it, add a task, exit, list in a fresh process and check reference/text/open state. Complete it, exit, list in a fresh process and check completed state. Add identical text and confirm distinct references. Replay the example interaction from UseCases. All invocations select the same file and have independent in-memory state.

This is ordinary-restart evidence, not power-loss or parallel-writer evidence.

Also prepare a valid collection through the real storage adapter with reference `18446744073709551616`. Through the real CLI, list and complete that task, then add another and require reference `18446744073709551617`. Restart and verify both references, texts and states round-trip without truncation. This spans parsing, allocation, formatting, storage and reload rather than relying on a domain-only large-integer test.

<a id="check-storage-errors"></a>
## CHECK-STORAGE-ERRORS — Storage failure preserves data

Type: Check. Method: storage contract tests plus real-process failure cases.

For unreadable, malformed, unsupported and invalid-record input, require a read failure, no success output and identical original bytes after every command. For a deterministic failure before publishing a write, require a write failure and unchanged existing bytes; when the file was absent it remains absent. Include a missing parent directory and verify no directory tree is created. Do not count a fake-only storage test as proof of real replacement behavior: at least one real adapter failure before publication and one successful replacement must be exercised. Use deliberate failure injection rather than assuming chmod semantics on every OS.

The suite documents which stages it covers. It does not establish crash recovery or filesystem guarantees beyond REQ-LOCAL.

<a id="check-structure"></a>
## CHECK-STRUCTURE — Responsibility boundaries

Type: Check. Method: source review, optionally supported by implementation-specific static analysis.

Inspect the realized component dependencies and Artifact ownership against the edge list. Confirm CLI parsing does not directly decode/write task files; task logic does not print user messages; file persistence does not decide offered actions. Check every realized file has one declared Artifact owner and that tooling manifests have been concretely accounted for. This is a bounded architecture review, not proof of every runtime path.

<a id="check-documentation"></a>
## CHECK-DOCUMENTATION — Realization instructions and change consistency

Type: Check. Method: independent document review and a clean-checkout walkthrough.

Follow the user guide to build/start the realized application and run its complete example. Follow the developer guide's actual test entrypoint. Check the agent entrypoint resolves to the canonical intent and development rules; compare changed behavior with affected guides and tests. Verify the offered scope has not expanded implicitly and default task data/build outputs are ignored. Record the candidate, relevant intent acceptance and review outcome rather than equating a test pass with owner acceptance.

This cannot run before realization. Link resolution alone does not establish content correctness.

<a id="check-overview"></a>
## CHECK-OVERVIEW — Intended user benefit

Type: Check. Method: human observation after realization.

Ask a person to record two tasks, complete one and identify the outstanding task from the list. Record whether the person can distinguish tasks/states and the difficulties observed. This is a bounded observation of REQ-OVERVIEW, not a quantitative productivity claim or an automatic acceptance decision.

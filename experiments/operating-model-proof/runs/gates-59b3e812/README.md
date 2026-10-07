# Frozen source 59b3e812 gate packet (draft)

This packet curates the Windows v5 and Linux v6 technical gate evidence for source commit `59b3e812a4da52cebba81d87d734e8e1cc6ab61b`. It contains sanitized receipt metadata and control summaries only; raw stdout/stderr, private environment files, and full temporary paths were not copied.

Windows v5 is a mixed result: the direct full Go suite passed with no timeout, while fixed-source `markitect verify` exited 1 at its `go-tests` check. Three Host tests failed because their fixture temp paths used a Windows short-name spelling where canonical path spelling was required. Nineteen skip events were recorded (17 top-level tests and 2 subtests); the two controller lease rename/swap tests skipped because Windows denied renaming their pinned parent directory. The verify command was the last gate; later gates were not run. Preserve this as an actual Windows FAIL, not as INVALID_SETUP or PASS.

Linux v6 passed with exit code 0 against the same exact source SHA, using Go's default timeout without override and network disabled. Its focused Host lease/writer controls and complete recordstore package ran verbosely and passed, including the controls that Windows skipped. This is a Linux technical pass only; it does not change the Windows v5 failure.

Linux v4 and v5 remain separate `INVALID_SETUP` attempts: v4 could not execute a helper from the Go build cache (`permission denied`); v5 could not find `python` on PATH for the projection fixture verification. Their source SHA and raw log hashes are preserved in the JSON; neither is rewritten or rescored.

No provider was invoked. These receipts establish gate behavior for one source SHA, not a capability-trial result, release decision, semantic assurance, or economic/human-attention claim. Later setup fixes must receive new run identities and be reported as new evidence.
# Independent plausibility review: stderr observability plan

## Assessment

The bounded corpus does not support a trustworthy causal stderr signature table. The plan's single alternative—a short-lived private capsule for local human classification—is proportionate to the stated goal of learning what a future message reports. It changes the discard-only privacy boundary, so it correctly requires a new explicit contract before implementation or execution. It should remain a reported-message category, never a verified root cause or evidence of historical equivalence.

## Evidence and claim limits

- The reviewed collector hash is `a9a1213146acbf31791e966992507b976ee6d3753bf50b27ecf14e7c42261df4`. Its level parser accepts a bare exact severity token after an optional ISO-Z timestamp, optionally followed by a fixed component token; it has no ANSI removal, bracketed-level normalization, or message-cause matching. It discards source text when reducing lines. Extending format recognition could only label severity/structure, not reconstruct a cause.
- The exact candidate hash `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68` contains the cited configuration-failure literals at the listed offsets. Nearby bytes point to multiple distinct code areas; they establish string membership, not which component emits the text to stderr, whether a full line is emitted, or whether that path is reachable during this diagnostic. The `[ERROR]` literal at offset 266074644 is embedded Python source for a missing `SKILL.md` message. These bytes do not prove the app-server's stderr format. The plan appropriately rejects broad substring predicates, including matches that occur inside unrelated words.
- The reviewed schema archive hash is `3157e8a55329cf4e5346676c3ef0c308402d2420f8f916d6a5b7e0e9c84356ad`; `v2/ErrorNotification.json` hashes to `eb702b074f6f8d6a7236e095b57fb24d69db183d3dc3ffab1954e65c2ead9e6d`. `CodexErrorInfo` and `TurnError` describe protocol payload shapes. They are not a serialization contract for stderr and do not identify an observed stderr cause.
- The reviewed client hash is `85a3181560c97e150d52fbb12458bea3958993ffaecbdcffa077ff14dc30bb5c`. Its stderr pipe feeds the collector and terminal event; stdout follows a separate frame parser. This supports the plan's distinction between schema-validated protocol data and free-form stderr.

Accordingly, static format or literal matching would invite false attribution. The plan's evidence does not establish that all possible cause strings are absent from the executable; it establishes only that the bounded evidence gives no emitter-and-format proof for the proposed predicates.

## Capsule boundary and minimality

The capsule retains only the first four complete stderr lines, up to 1,024 bytes including delimiters, because R2 observed four complete lines totaling 765 bytes. That is a finite observation budget, not a prediction that a future attempt will repeat those bytes or produce a useful category. Excluding a line cut by the byte cap and stopping retention after the first four lines avoids searching for a more interesting excerpt. The existing first-stderr terminal rule remains in force, and collection stays within the existing bounded cleanup.

Local human classification is the minimum semantic step that can map actual message content to the plan's small enum without sending raw text to a model or inventing automatic signatures. The five categories describe the report in the captured text. `other_or_insufficient`, line number, review completion, and capture completeness preserve ambiguity and truncation; none should be interpreted as proof of the underlying cause. The sanitized receipt should contain only those fields, never the message or a content fingerprint. A missing or late review must expire as specified, with no repeat.

Current-user DPAPI and a named human reviewer describe an at-rest and operational boundary; they do not, by themselves, establish reviewer-exclusive access from other processes running as the same user. Keep that limit explicit. Any implementation should expose plaintext only in the named local review path, avoid plaintext files, console/tool output, clipboard, Git, model input, and callback payloads, and destroy the encrypted capsule after review or by the 24-hour expiry. If that local boundary cannot be maintained, the plan's stated stop condition—do not launch—should apply.

The proposed next step is appropriately limited to a separately authorized additive implementation plus focused offline privacy/terminal validation, with zero runtime quota. No capture, implementation, tests, or execution was performed for this review.

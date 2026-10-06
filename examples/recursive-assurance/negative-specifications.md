# Negative specifications

These cases define failure expectations for a future bounded Executor/Verifier experiment; they are not claims that an agent was run.

- If an OrderRules implementation accepts zero or negative quantity, accepts a negative price, or overflows line multiplication, the Orders check fails.
- If Billing accepts negative amounts or silently wraps addition, the Billing check fails.
- If Checkout bypasses Orders validation or drops the accepted line total when calling Billing, the Checkout parent check fails while the independent Orders and Billing checks can pass.
- If Commerce invokes only one Checkout, loses either Checkout amount, omits previous outstanding amounts, or bypasses the shared Billing overflow behavior, the Commerce parent check fails while all child checks can pass.
- If a source Kind, typed reference, Definition, selected descendant, policy, package pin, runtime binding, target root, or exact required check is unknown, missing, ambiguous, or stale, fixed-source preparation must reject or escalate; it must not guess a filename or substitute a sibling scope.
- If a ProjectionPolicy changes source semantics without a reviewed source/package change, the exact source/policy and package digests change; prior candidate or verification evidence cannot be reused as fresh.
- Missing candidate source is incomplete evidence, not PASS. Passing package compilation alone is not parent-composition evidence. Executor-authored tests do not replace the independent configured checks.
- A real AI result requires a fresh configured Executor and separate Verifier bound to the exact committed source and candidate bytes. Prepared requests, this test suite, local synthetic candidates, and historical C9 FAIL remain distinct and cannot be relabeled as that result.
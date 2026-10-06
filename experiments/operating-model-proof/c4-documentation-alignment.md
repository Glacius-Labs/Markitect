# C4 documentation alignment: separate trial contract

This amendment defines a new trial, not a repair of recorded C4 inputs or results.
The original fixed Documentation check and all other fixed checks are preserved.

## Evidence and diagnosis

In `c4-module-proposal-v5-02`, fresh Executor artifacts and the deterministic
Markdown were applied. The Orders, Billing and composition checks passed. The
Documentation check failed because `TryCalculateTotalCents` and
`SumOutstandingCents` were absent. The fresh Product Verifier reported the three
selected business/documentation clauses fulfilled, and failed the policy
observations solely because the fixed Documentation check failed.

Those names are declared in canonical .NET ProjectionPolicy Definitions, but
neither was a selected Markdown source. A check cannot silently introduce a
canonical documentation requirement. This is an adopter selection/check mismatch;
it is not evidence that a semantic Markdown projector is missing.

## New exact input

Use `examples/operating-model/canonical-documentation-aligned.yaml`. It selects a
new Product Projection file with the same identity/target and the same three
business Definitions, plus the two existing .NET ProjectionPolicy Definitions.
A new explicit policy projects those selected representation contracts to
Markdown. No method name is duplicated into a second canonical rule. Foundation,
Commerce and target Module pins, the business/API owners and all four check
commands/implementations are unchanged. The original source remains executable.

The setup test proves original Markdown lacks both identifiers and new Markdown
contains both by selecting those existing owners. It also compares unchanged
canonical/check/module inputs. This is setup evidence only.

## Required fresh trial

Before any provider invocation, freeze the full immutable new fixture revision,
selected input paths/bytes, source binary/build receipt, driver/runtime identities,
exact checks, absent external ledger and independent private-log directory. Use a
non-protected `codex/` branch and canonical absolute paths. Bind one trial ID
through Propose, Execute, reviewed Apply and fresh Verify. Do not precreate an
empty ledger leaf. Preserve failed attempts and final output exactly.

Run two fresh leaf Executors, deterministic Markdown and three independent fresh
Verifier sessions with the unchanged four checks. Apply remains
`materialized-unverified`; only the actual immutable verification result determines
the trial outcome. No owner acceptance or economic/productivity claim is implied.
A failure is retained and diagnosed before any subsequent amendment.

The broader queued protocol originally prescribed sequential C3-C13 execution.
Some source-bound trials were run in parallel to implement independent accepted
capabilities. That sequencing deviation is explicit; individual receipts do not
establish conformance to the old program's timing/order.

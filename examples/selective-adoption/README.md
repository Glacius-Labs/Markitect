# Selective adoption handoff

This fixture exercises the explicit boundary between selected source evidence and a proposed reusable rule. It is a small, synthetic Git repository created by the end-to-end test; the test builds and runs the real `markitect` CLI. No internal Go records are assembled by the fixture, and no provider or account is involved.

The flow is:

1. `prepare --scope ... --output ...` previews a handoff from exact Git paths at a fixed commit. The preview does not create the destination.
2. `prepare ... --write --expect <handoff digest>` stores the exact selected evidence in a new external directory.
3. `copy-me --workspace ... --queue ... --decision ...` validates supplied support, counterexample, conflict, coverage, uncertainty, and decision records. A successful report still says `adopted: false`.

Run the full CLI proof from the repository root:

```powershell
go test ./examples -run '^TestSelectiveAdoptionCLI$' -count=1
```

The test covers preview without external writes, unchanged source state, exclusion of unselected/private fixture content, handoff creation, validated-but-unauthenticated review, stale expected digest refusal after selected bytes change, and a new capture after an unselected-only commit whose selected snapshot digest remains stable.

## Public report replay

`replay.go` accepts an already built Markitect executable and a local Markitect checkout. It supplies one fixed revision and exactly one public path, `docs/validation/parallel-wave-konfyra.md`; it does not inspect the checkout itself. The replay is optional because a normal clone or CI checkout may not contain the fixed commit. It creates a temporary external handoff, validates a clearly synthetic Copy Me candidate/decision bound to those exact bytes, then removes the temporary workspace on exit.

```powershell
go run ./examples/selective-adoption/replay.go --markitect C:\tools\markitect.exe --public-repo C:\src\Markitect
```

The replay leaves the source checkout unchanged; it writes the handoff and interpretation records only under a temporary directory and removes that directory when it exits. It prints the handoff identity and the single selected path, but no captured report text. The fixed commit is `93181bb9bc1af0e663b3daa1a4ff772822320307`. The report is public and sanitized; this is evidence only for the selected file and does not imply inspection of the rest of that repository or validation of its claims.

## Limits

This proves explicit selection, fixed-revision byte capture, integrity checks, and separation of interpretation from adoption. It does not prove that the selected evidence is representative, that a proposed rule is correct, that the reviewer is authenticated, or that a validated decision changes Markitect policy. Coverage records describe only the supplied questions. Hashes bind bytes; they do not establish truth, authority, privacy at a remote service, or completeness.

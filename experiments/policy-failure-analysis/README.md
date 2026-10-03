# Policy-failure analysis replay

This isolated replay exercises the policy failure analysis workflow designed in [the design note](../../docs/design/policy-failure-analysis.md) against the preserved MyMeetings architecture-contract history. It writes results only below this directory and uses a disposable checkout under the ignored `.artifacts/` directory. The earlier adoption pilot under `experiments/real-project-adoption/` is input evidence and must remain unchanged.

The replay uses the exact bundle SHA and the v1, failing-v2, first-Validator, waived, and final revisions recorded in `replay.ps1`. Both pinned package archives are tracked in the bundle snapshots; the script verifies their exact recorded SHA-256 values and never repacks or rewrites them.

After building the candidate executable from the source commit being evaluated, run:

```powershell
pwsh -NoProfile -File experiments/policy-failure-analysis/replay.ps1 `
  -CandidateExecutable C:\path\to\markitect.exe `
  -CandidateSourceCommit <full-source-commit> `
  -ExpectedCandidateSha256 <64-character-executable-sha256>
```

The replay captures each command's stdout, stderr, exit code, and fixed checkout tree identity. It expects diagnostic Context and Impact to emit completed analysis while returning exit code 1 for the failing v2 candidate, and expects strict Check/Context/Impact to keep their previous failure behavior. Intermediate and waived states are included to show the existing staged evolution. No reconciliation, rendering, application-code edits, exception edits, or projection writes are performed. The JSON run record substitutes `<replay-checkout>` for the exact temporary checkout path in command argument metadata; actual invocations use the path on disk.

The script fingerprints every regular file in the disposable checkout before and after each Markitect invocation, excluding only `.git`; this also detects changes to ignored files and projections. It verifies the candidate's Go build metadata against the supplied source commit and requires `vcs.modified=false`. Recorded checkout and executable paths are normalized to placeholders; build-info line endings are presentation-normalized by removing trailing spaces/tabs and keeping exactly one final newline. The commands run against their actual paths, and the Go metadata values are preserved. The optional `PublicV012Executable` parameter checks the old published binary's identity only. It does not run that binary as a behavioral baseline; the preserved pilot captures remain the baseline evidence.

Each run gets a new timestamped directory under `results/`. A run record binds the bundle, candidate executable digest, source revision supplied by the operator, pinned package digests, fixed project revisions, command exit codes, and clean-tree checks. Interpret this as workflow inspectability evidence only; it makes no productivity, correctness, safety, or runtime claims.

The first captured run is documented in [report.md](report.md). It passed this MyMeetings replay for candidate `1549608438f9c0d24d29d43608515bce388c14e7`; an unrelated Impact result-scope issue was found afterward, so this run is evidence for that candidate and is not the final source-slice validation. The report preserves its exact inputs and outputs. Repeat the replay against the corrected candidate before treating the implementation as complete.

# T9 what a verdict covers: native checks — 30 October 2026

Native checks for step T9 of the [delivery plan](../implementation-plan.md#concept-20-build) ([DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)): every maker's and checker's task now says what a verdict covers. That is the model in full for each item, with evidence for each example, and the code of the Manager's own Package only. A parent integrates its children's work by using it. Both runs repeat a failure the decision names, as checks, not counted data. They fall under the owner's standing run approval of 28 October 2026 for build runs under a dispatch ([build workstream, Decisions](../workstreams/build.md)). The pilot's c3 result stays a stop failure for the 2.0 arm, and nothing here counts.

## Questions

- **Q1, the contradiction:** does the pilot's plant `makerspace-c3-discount-contradiction` now end in Root's model-defect escalation, with nothing landed? In the [T7 pilot](t7-pilot-20261029.md) the 2.0 arm landed it. The plant's bulk-discount table says 8 % for 10 to 24 units, while the same Definition's worked example uses 5 %.
- **Q2, the own Package:** in the Codex Library M1, does no checker reject its Package's work because a parent's file does not use it yet? In the [Codex Library smoke](codex-library-smoke-20261030.md), `lending`'s checker did that three times, and nothing landed.

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `dev/t9-verdict-scope` at its reviewed head `SOURCE`. `markitect.exe`, `markitect-remote` (Linux, amd64), the Manager image, the Go Manager image and the sandbox image are built from it under the lock |
| Q1 command | The pilot's runner (`t7/pilot.py` in Build's notes folder, unchanged) with a one-unit configuration: the plant `makerspace-c3-discount-contradiction`, arm M only, from the C2 reference start with its registry pages. That means `suite.py run --native` with [markitect_arm.py](../../tools/evaluation/README.md#the-concept-20-arm), then judging with the plant's judge. Claude `claude-sonnet-5-5`, effort `high`, deadline 5400 s. The suite takes the lock itself |
| Q2 command | `t5smoke/smoke.py` in Build's notes folder, unchanged, as in the Codex Library smoke: a fresh project from the Library M1 with one registry page per Package, `apply --full --deadline 60m`, then `verify --full --deadline 30m`, then build, vet and test of the landed state. Codex `0.162.0`, `MARKITECT_PROVIDER=codex`, `MARKITECT_MODEL=gpt-6-luna`, effort `high` |
| Login | Claude for Q1 and Codex for Q2, the account login only, never an API key. Every `ANTHROPIC_*`, `CLAUDE_*`, `OPENAI_*` and `CODEX_*` variable is removed, and each runner starts detached with a clean environment |
| Limits | The host's: 6 maker and 6 checker starts per Manager, one hour per start; each Manager container 1 CPU, 2 GiB, 256 processes; the remote 2 CPUs, 4 GiB, 1,024 processes. One launch per question; a second only after a fix of a defect the first shows, as an amendment here |
| Engine | Docker Desktop's regular engine. Before each launch, `docker ps` lists no container that is not Markitect's and `tasklist` shows no `dotnet` process. **Build clears the window with the owner before each launch** |
| Lock | `NATIVE-RUN.lock` (`local_cbc94143…`): taken exclusively for the image builds and released, then taken by the suite for Q1 and by the smoke harness for Q2. Nothing else wraps a runner |
| Abort | Build's word, a failed login, a quota stop, a credential found in the evidence, or a Run past its deadline |
| Cleanup | The host removes each Run's remote, network, containers and volumes; afterwards nothing labelled with a Run may remain |
| Evidence | Outside the repository, `C:\Users\Consiliari\AppData\Local\markitect-t9\`: the Results, the remote's decision logs, the transcripts, the judgment for Q1, the post-checks and a scan of every file for the login's tokens |

## Pass criteria

- **Q1:** the Run lands nothing and ends with Root's model-defect escalation (an approved `model-problem` verdict or an escalation Root settles as a model defect). The escalation names the contradiction between the table and its example.
- **Q2:** no checker's review disputes a verdict because a parent's file does not use the Package's work yet. Whether the Run lands is recorded, but it is not this check's criterion.

## Results

Pending.

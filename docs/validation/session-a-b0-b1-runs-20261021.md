# Session A B0/B1 runs, 21 October 2026

**Historical record.** These are the provider runs Session A made before the owner returned to [one container per Turn](../design/canonical-model/manager-apply-workflow.md#per-turn-container-baseline). B1, the reuse of held per-Manager containers integrated by #572, was removed again afterwards. This page compares nothing and claims no speedup. Each attempt stays bound to the build it actually ran on.

## Builds and inputs

| | B0 | B1 |
| --- | --- | --- |
| Product (Main) | `b9cdd6d893fa9487a09016a3520c3c3442660c90`: direct/staged, context delivery, mandatory Model assessment, closed Runs, observation kinds; fresh container per Turn | `fd94e365f9c1c060c9419be67b70ca072b6c1128`: B0 plus held per-Manager containers (#572) |
| CLI sha256 | `4d472cfb…5432ab` | `89a69951…3190a6` |
| Runtime image | `sha256:ab74447e384ff86b0dc4c4abd852fa3030e904496af72ae93d294350613efbc1` | `sha256:17cf64c4054302e81678603192437def0e3073bfecfd0a784208bdda8817966d` |
| Agent image | `sha256:8d1972a4ed4e02b53c5b6687b469d8ce38e0bf0d503efa5dc6edd24662fe88ed` | `sha256:388a040c3abe60eb073d3a699b4e6ae9c1f20ee2e3e7f219e08f41a4b7a6a29b` |
| Python image (oracle) | `sha256:5f478c0facac4aeec7152046545c3af3a1a39762191a2d6a10ad67fc242b37d5` | `sha256:2c95577fcc70008dc21dbcad83bee80df23f61cc5211f7ae7f4a47c7170cd9b9` |
| Host profile sha256 | `5b7082a2…b1da3c885` | `839e46bf…f5164cc` |

Both builds used the same kit: `experiments/apply-trials` at the build's own commit, copied with stockroom's and depot's `strategy` set to `direct` (the owner's provisional baseline). Kit digest `45f005c3054abb3d35222663b081ba57a1c72bd1ce0c52e0c53a0d5108971f7f`. The model was `claude-sonnet-5-5` with effort high for every Execute and Verify Turn. The harness was `experiments/apply-trials/run/run_case.sh` of the build's commit.

## Attempts

Every start counts against the original Sonnet allowance, including those that never reached a provider.

| # | Attempt | Build | Case | Outcome | Class |
| --- | --- | --- | --- | --- | --- |
| 1 | SA-B0-SN-1 | B0 | S-N | No run: the copied `run_case.sh` did not find `patch_runconfig.py` beside it | Session A harness error; no provider use |
| 2 | SA-B0-SQV-1 | B0 | S-QV | Same harness error | Session A harness error; no provider use |
| 3 | SA-B0-SN-2 | B0 | S-N | `not_completed` in preparation: "observe image path /work: context deadline exceeded" | Infrastructure. Session A's own Linux checks were loading the same Docker engine; no provider use |
| 4 | SA-B0-SQV-2 | B0 | S-QV | **Failed by the Model assessment**: Root confirmed `model_gap` (the missing maximum name length). No work Pass, nothing published, 0 changed paths; 176 s, 6 Turns, $0.99 | Correct under the closed-Run policy |
| 5 | SA-B0-SN-3 | B0 | S-N | **Verified**, publication `no_change`: assessment satisfied, Implement satisfied without change, Verify satisfied on the unchanged tree; 731 s, 18 Turns, $3.61 | Correct: conforming stand preserved |
| 6 | SA-B1-SN-1 | B1 | S-N | `incomplete`: "capture of the stand at the end of p2 is uncertain: workspace volume remains mounted by 3 container(s)"; 371 s, 12 Turns, $1.36 | **B1 product defect**: the kept containers still mounted the work volume at the capture. The capture guard held, and nothing was published |
| 7 | SA-B1-SQV-1 | B1 | S-QV | **Failed by the Model assessment**, `model_gap` as in #4; 134 s, 6 Turns, $0.58 | Correct |

**What the outcomes mean**
- **Model-only acceptance** happened where the assessment sealed `satisfied` (#5, #6). It establishes no realization.
- **Artifact verification** is only #5's clean Verify of the candidate.
- **Technical incompleteness:** #3 and #6 are neither a model conclusion nor an artifact result.

The reuse itself worked in #6 before the capture refused:
- Reviewers' and Managers' instances were reused across the Turns of a Pass.
- Root got a separate read-only view for its coordination Turns.
- New instances started in 0.5–0.9 s, and every release was observed.

A tested fix that released the work environments before each capture (#573, branch `fix/assessment-reason-clip`, `630651c5`) was closed unmerged when the owner decided for one container per Turn.

## External scoring

The kit's `assess.py` did not score these runs correctly against the product as it then stood:
- For S-QV it reported `met: false` while naming the defect (`defectNamed: true`). Its accepted stop reasons do not yet include the Model assessment's `the model assessment confirmed a material model problem:`.
- For S-N (#5) it stopped with `AttributeError: 'str' object has no attribute 'get'` in `summarize_report`, on the shape of the report's preparation diagnostics.

The outcomes above are Session A's own reading of the reports. Session B owns the oracle; its scoring was not repeated here.

## Evidence

Every attempt directory stays under `~/mk-sa/` in WSL Ubuntu. It holds the binding, timing, the Apply report, events, transcripts, the materialized repository and the kit copy. An archive without the token directories is under `~/session-a/evidence/`, together with the ledger and the build bindings:

| File | sha256 |
| --- | --- |
| `b0b1-attempts-20261021.tar.gz` | `038d314634e191c5ed6e5de6027fe8daf80d73bdee12d2d87e70f3ddfa4ab75e` |
| `ledger-b0b1-20261021.txt` | `b840473a9648752a3359234d474ccd2f61063d8f29f78d1ac89d97e3cccd28ae` |
| `builds-b0b1-20261021.txt` | `54c78d5c11ce8efe1fb7e02e11301cc2be83553839c8253dff072e250261b62b` |

Not shown here:
- Whether the assessment method finds gaps without false alarms on other models (Session C's series).
- Any comparison of strategies or lifecycles (Session D).
- Any run of the restored per-Turn build.

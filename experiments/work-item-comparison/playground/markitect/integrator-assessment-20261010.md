# Integrator runtime assessment, 2026-10-10

Direct human request: inspect the Integrator chat and current Markitect acceptance
run, assess problems, and provide concrete help. This is a read-only product
assessment; only this Scientist record was added. No product edits, provider
starts, credential access, configuration changes, or Conventional repetitions.
Classification: implementation-only research evidence, no product-intent change.

## Bound observation

- Integrator chat: `01a121a0-0417-71c1-9e4b-be742f8c1146`, local.
- Frozen product source: `5ad9e069ee6252ba9a9dd66b8986383ecec3b7f4`.
- Binary SHA-256: `88234adb116d7c8b0b6c3eb35ece88704d66c6ac33d7b05aca3aeb4e49efa6ea`.
- Batch: `a01-verdict-fb7b6fe1d6fe456fa94c9321624d2779`.
- Plan: `3a021f942d3b80f287ec9000d2af1632`; state revision 9 is failed.
- Observer `d8fe20710f634e65aaf993885e406ce8` exited 2 at
  `2026-10-10T04:15:52.3300071Z`, before the trusted recovery boundary.
- Runtime requested Luna/high, `:workspace`, approval `never`, inherited
  environment, MXC. Effective backend remains unknown in its recorded binding.
- Native executable: stable `codex-cli 0.162.0`, SHA-256
  `dce685d569526ef4712fa82d20ff20e1309baad5fcd34215e1126698f611242f`.

This attempt has no successful child delivery, Verify, Apply or recovery proof.
An allocated batch label does not override the terminal observer and run state.
The active Integrator was already editing process/session/review guidance when
inspected; those uncommitted changes are not part of the frozen attempt above.

## Findings and repair direction

1. **Actual shell CWD differs from recorded CWD.** Latest Root journal sequences
   81 and 152 report the Host workspace in `item.cwd`, but command output resolves
   relative reads and `Get-Location` to `C:\Windows\System32`. Reviewer sequences
   81 and 203 show the same discrepancy. Receipt echoes cannot establish actual
   shell location or effective write access. Frozen `process.go` already sets
   `cmd.Dir`; launch arguments already include `allow_login_shell=false`.
   Missing launch flags are therefore not a demonstrated cause.

2. **A proven native edit path exists.** Previous attempt's Root journal
   sequences 65/82 also suffered the CWD error. Sequence 100 explicitly changed
   location to the original Host workspace and read its context; sequence 106
   completed a native `fileChange` updating that workspace's README. The latest
   Root instead switched to `AppData\Local\Packages\...\LocalCache` at sequence
   169; its shell write at sequence 213 was denied at that alternate path.
   This supports using the native editor against the exact Host-owned path and
   preserving artifact harvest. It does not prove the low-level cause of the
   CWD deviation or establish that every shell write at the original path fails.
   Location guidance is also needed for read-only Reviewer/Verifier calls;
   current writable-role-only guidance does not cover their observed issue.
   Do not rewrite ownership, aliases or permissions to make another path writable.

3. **Shell exit code alone hides this failure.** The denied `Set-Content` command
   is recorded with exit code 0 because its PowerShell error was nonterminating.
   Actual harvested bytes and check results must govern artifact acceptance.
   A command-level success label is not evidence that README changed.

4. **Review failed its reference contract, not its unchanged-file scope.** The
   supplied candidate contains the unchanged TODO README. Its allowed grounding
   references include `artifact-path:README.md` and the exact
   `statement:["project.markitect.example.org/v1alpha1","Statement","","repository-overview"]`.
   Reviewer sequence 345 correctly identifies the missing documentation but
   places explanation prose in `grounding`. Frozen `review.go:645-660` accepts
   only a same-file reference token; `:513-557` includes existing owned files
   even when unchanged. Clarify exact-copy `path`/`grounding` and put explanation
   in `expectation`; preserve the validator. Independent static review agrees.
   Missing-file/tombstone representability is a separate potential edge, not
   the cause of this attempt, and needs its own concrete regression if addressed.

5. **Earlier startup and report failures are distinct.** Prior Windows ACL/setup
   failures and child-environment folder failures do not explain this already
   started turn. The previous acceptance attempt reached successful Root/Docs
   reviews and actual overlapping source/helper work, then rejected a Source
   review with `pass` plus positive findings. The subsequent actionable-findings
   clarification addresses that separate contract error. Do not relax semantic
   guards or classify all these failures as one Windows startup problem.

The Integrator's observed editor/Host-path and exact-grounding guidance changes
address the two current mechanisms. A focused contract regression and a bounded
actual run remain necessary to demonstrate effectiveness, especially for
Reviewer/Verifier shell location. Native completion alone is not A01 acceptance.

## Evidence locations and preservation

All acceptance paths below are relative to the Product-owned root
`C:/Users/Consiliari/AppData/Local/Temp/markitect-a01-3c5c733388014645b0ca178338078b6a/`.
Raw records were read in place, not copied into the research repository.

| Record | Relative path | File SHA-256 |
|---|---|---|
| Latest state | `repo/.markitect/runs/3a021f942d3b80f287ec9000d2af1632/states/00000009.json` | `1cc3c9564a35ca5e376a732d36b42fc43fd234af582dffc09e98ff42e874ad6e` |
| Latest Root journal | `repo/.markitect/runs/private/codex-app-server/a0fa2ea044c51c45d23df683f4cf04e6b813584ba26f0f7f75fd2fdc30026c86/da8db970094f16e854d11ee6ada437cb/events.jsonl` | `619c73997adc6c43e0c8fdaabab1dcc86643db9811f708b01d6bb7380e29c2c4` |
| Latest Reviewer journal | `repo/.markitect/runs/private/codex-app-server/e6db56c93c9e735fd3efa6f1e2316d8ce247b59f03bfb5f686493d27efba4ef7/862ae730cbcb21ec70c31f4182fa861f/events.jsonl` | `6fd9bc5b33d97091507b757a577d2efcba790314fc9cb7a1c7e2df0ece46952c` |
| Previous Root journal | `repo/.markitect/runs/private/codex-app-server/d619466febab0d4e274e39e80be72f343dfa43149158e48718c134cead85683a/336d7df7a58cf874568188ee448a2d3c/events.jsonl` | `d92768276845b03694776db6146da300d4ec1ede9ae72a9df983d4254b47d2f5` |

These are whole-file hashes, distinct from provider receipt event-chain digests.
Frozen source lines above are from product `5ad9e069`, not moving WIP line numbers.

The sanitized [Windows runtime handoff](windows-runtime-handoff-20261010.md)
describes our earlier successful alpha.2 executable and wire shapes. Its binary
differs from this stable inner executable; it is corroborating history, not a
blind binary replacement prescription. No private Scientist case/holdout inputs
or assessment solutions were shared with the Product owner.

Official [App Server documentation](https://learn.chatgpt.com/docs/app-server)
documents per-turn CWD and output schemas. The official
[configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
documents non-login defaults for `allow_login_shell=false`. Neither establishes
the specific package-path/CWD cause observed here or guarantees that the flags
repair it; the retained native outputs above supply the incident evidence.

Concrete findings and the sanitized handoff were sent to Integrator under the
direct human help request. A subsequent compact chat snapshot confirms Integrator
is extending the CWD guidance to Reviewer/Verifier and independently reviewing
the changes while retaining permissions and validation. This confirms uptake,
not successful execution of a corrected candidate. Product fixes and subsequent
native acceptance remain with Integrator. This assessment establishes no method
winner, Main/release readiness or human acceptance.

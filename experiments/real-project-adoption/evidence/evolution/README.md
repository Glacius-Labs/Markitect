# MyMeetings package policy evolution

This evidence directory contains raw `.stdout.txt`, `.stderr.txt`, and `.exit.txt` files for the captured commands. Commands ran from `.artifacts/adoption/evolution-final`, except the package and upstream inputs below are identified from the frozen fixture and its setup records.

## Frozen inputs

- Public MIT adopter: `https://github.com/kgrzybek/modular-monolith-with-ddd`, commit `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`; 646 selected source files / 711,461 bytes.
- Frozen v1 adopter: `fd94a689aab857704c3a4a45ac2abe0fc0e3185c`.
- Public Markitect binary: `../public-v012/markitect-v0.12.0-windows-amd64.exe`, SHA-256 `b03424560caa460322e3580785d6abc9dfbf2137df878e03ddedcf76b3a8fa47`.
- v1 archive: `.markitect/packages/mymeetings-architecture-1.0.0.zip`, source revision `76584ffcf25c83876b32e5567ba4cd1351149eb5`, SHA-256 `0c7395055aac7f4c35afcff404481a77e2ce6475c863a435afc2586dd388b1cc`.
- v2 archive: `.markitect/packages/mymeetings-architecture-2.0.0.zip`, source `git:e0ae3c5181323e0d2a03312524010cc817652330`, SHA-256 `0ed97c9f0b21e10b5e2a0ff232bdb910bd468ac04f84441ae24d5b7cefb2b504`. These values match `evidence/setup/package-pins.json` and `package-2-pack.out`.

## Results

At frozen v1, fixed-snapshot replay `check` passed twice (exit 0), with identical stdout SHA-256 `eeb8e6898f2d751f682b0a46aa38ba824915d7e18b33c967dfe86115c55a60fc`. The fixed model passed with digest `sha256:d115aeef1780a8bdf9aaec2dca2d67c696b4892740c7e59830d6dcd035c89e94`; both selected UseCase contexts passed. Neither packaged archive was rewritten; both current archive hashes still equal the setup pin records above.

The exact package v2 pin is commit `a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f`. At that fixed SHA, `check` and `model` each failed twice (exit 1) with byte-identical repeated stdout: check SHA-256 `4075f6ca0e20171fdf17dc2190da4b8b5fb91a2ba2d0901a26796559729cf1fa`, model SHA-256 `1d30c3fd96bab0ae37c0bb2a30d750506a3fb850dd932362f1bff81a0ed77893`. The only failed PolicyResults were `selected-commands-require-validator` for `UseCase/add-meeting-attendee` and `UseCase/cancel-meeting`. Both `selected-validation-cohort-is-command` results passed. The fixed model reported `validationStatus: failed`, model digest `sha256:9e03161abf413beacf9ddc47fa92033032bc2a93990301b6f4cb7ddb8ee6beb7`.

At that invalid pin, fixed `context` requests for the two selected UseCases and `impact --base <v1> --revision <pin>` exited 1. Their output is the generic failed check report: it contains policy diagnostics but no context `entry` or impact `base`/`affected`. These failed PolicyResult subjects are not an Impact-identified set. The exact refusal output is preserved in `v2-pin-context-*` and `v1-to-v2-pin-impact*`.

The staged transition committed the attendee Validator first at `3e138c146736dad1ddb11dc7c509a61d95ab4398`: attendee passed and cancel alone failed. Commit `65c1d959e6f5d1b866effbc5188c353c82b2405d` records one expiring, digest-bound exception for subject `architecture/architecture.mymeetings.example/v1alpha1/UseCase/cancel-meeting`, constraint `selected-commands-require-validator`, constraint digest `sha256:8d4e312bdb9a2f5569371b9a6be101ee76a9ea0d4007b0087e441031d248f2a8`, and subject digest `sha256:cb2171eb85c210db94ac2a0d6bc7eebabaf9b83cb2f5e7b55ffcbae99dbf05cd`. It is dated `2026-10-03`, expires `2026-10-04`, and explicitly records that its metadata is not an approval. The result is `waived`; fixed contexts for cancel and unrelated `get-meeting-fees` both exit 0. The intermediate `check` still exits 1 because generated views were stale; its output has the waived result and generated-output drift diagnostics. The intermediate impact exits 0 and includes a global `configuration` cause. The temporary exception is removed in the final candidate.

The converged adopter candidate is `c507172b24da9005904422c03cc3d66a2b5efcec` on `codex/architecture-policy-evolution`. It includes both FluentValidation validators, two Validator resources and explicit UseCase references, removes the exception, and refreshes Markitect-owned projections. Fixed `check`, `model`, and both selected contexts pass. Check and model were each repeated with identical output: check SHA-256 `13d4bc3327d5497512604176ea922cc6270d865eca9724be491b68af0a683159`, model SHA-256 `550ef40c1cf6df401f47a87038bb53858081b6b89112fa7b742c94cd44e92277`. The fixed model reports `validationStatus: passed`, model digest `sha256:3398b29cf0b3bf88a432fe79506bed87f5f2afccd859991db618857fcdab80a3`.

Final immutable impact from v1 to the converged candidate exits 0. It lists 73 affected GraphKeys and records configuration, domain/package and inventory changes, generated outputs, and declared inputs. It includes all 12 UseCases. Against the selected validation cohort, 2 are in the known UseCase affected set and 10 other UseCases are included by conservative invalidation: `accept-proposal`, `authenticate`, `create-meeting`, `create-price-list-item`, `get-meeting-attendees`, `get-meeting-fees`, `get-meeting-group-details`, `get-member`, `propose-meeting-group`, and `register-new-user`. This count is at UseCase granularity; it does not treat every context input as a code change or infer source semantics. The intended set is the two selected commands, their Handler/Validator surfaces, and the package policy resources.

A separate temporary comment-only probe changed the declared `AddMeetingAttendeeCommandHandler.cs` input from final SHA `c507172b24da9005904422c03cc3d66a2b5efcec` to probe SHA `fdc32ac004d17e3b4e91443b7c59fe035ce0d7b3`. Impact changed one path and affected exactly `architecture/architecture.mymeetings.example/v1alpha1/Handler/add-meeting-attendee` and `architecture/architecture.mymeetings.example/v1alpha1/UseCase/add-meeting-attendee`. The probe commit was removed by resetting to the final SHA. The handler file SHA-256 before and after is `328e756629a140bbdf5c92aaa51f5ab6b172d2e36fa18dfc5d54dcecb3b9673c`; the temporary version hash is in `handler-comment-probe.sha256.txt`.

The two source validators add only `MeetingId.NotEmpty()` rules for `AddMeetingAttendeeCommand.MeetingId` and `CancelMeetingCommand.MeetingId`. The Meetings Application restored from only `https://api.nuget.org/v3/index.json` and built with `--no-restore`; both exited 0. The build reported 0 warnings and 0 errors. This verifies compilation, not runtime validator registration or execution. No tests were run.

## Fixed command forms

All Markitect commands used `../public-v012/markitect-v0.12.0-windows-amd64.exe` and `--repo .`. Baseline used `--revision fd94a689aab857704c3a4a45ac2abe0fc0e3185c`; the invalid exact pin used `--revision a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f`; the converged candidate used `--revision c507172b24da9005904422c03cc3d66a2b5efcec`. Commands were:

```powershell
$tool check --repo . --revision <revision>
$tool model --repo . --revision <revision>
$tool context --api-version architecture.mymeetings.example/v1alpha1 --kind UseCase --name add-meeting-attendee --namespace architecture --repo . --revision <revision>
$tool context --api-version architecture.mymeetings.example/v1alpha1 --kind UseCase --name cancel-meeting --namespace architecture --repo . --revision <revision>
$tool impact --base fd94a689aab857704c3a4a45ac2abe0fc0e3185c --repo . --revision <revision>
$tool render --repo . --write
dotnet restore src/Modules/Meetings/Application/CompanyName.MyMeetings.Modules.Meetings.Application.csproj --source https://api.nuget.org/v3/index.json --verbosity minimal
dotnet build src/Modules/Meetings/Application/CompanyName.MyMeetings.Modules.Meetings.Application.csproj --no-restore --verbosity minimal
```

Every captured invocation has its raw streams and exit file. The temporary Handler probe command and immutable SHAs are in `handler-comment-impact.stdout.txt` and `handler-comment-probe.revision.txt`.

# Windows CLI onboarding exercise

This dated exercise measures a machine-run path through a published Windows CLI. It is a reproducible adoption smoke exercise, not novice-user research: one maintainer/agent runs a fixed synthetic task, and the timer does not represent a population or a human learning curve.

## Install and identify the release

For current installation, follow the pinned Windows instructions in the repository [README](../README.md#install-markitect). To reproduce the dated exercise, use the immutable `v0.5.0` release instead: set `$tag = 'v0.5.0'`, `$sha256 = '6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178'`, and the expected version string to `Markitect 0.5.0 (windows/amd64)` in those instructions. Confirm the installed path, hash, and version before authoring:

```powershell
$binary = (Get-Command markitect.exe -ErrorAction Stop).Source
Get-FileHash -LiteralPath $binary -Algorithm SHA256
& $binary version
```

The release binary is already installed when the measurement script receives `-MarkitectBinary`; the script records, rather than silently substitutes, its observed version and digest. The `install` command in Markitect is for pinning the CLI distribution into an adopting repository and requires the matching release bundle; this exercise's install step is the documented Windows package installation above.

## Run the bounded task

The synthetic [parcel-support fixture](../examples/onboarding/delivery-service/README.md) contains shared customer-data guidance, a policy, workflow, contract, agent, and Skill entrypoint. `tasks/query-only-prompt.md` with `tasks/query-only-expected.yaml` is the shared read-only task for MCP/CLI find, explain, and context parity. `tasks/policy-edit-prompt.md` with `tasks/expected.yaml` is a separate CLI-only mutation task that also exercises verification and fixed-SHA impact. Oracle and prompt files are excluded from the measured project's Git snapshots.

The scripted replay of this task, `scripts/onboarding/Measure-Adoption.ps1`, was retired on 11 October 2026 (backlog CI-03). It mixed the model-first `init` with legacy commands. Two checks now cover the model-first onboarding path:
- On every pull request, the [smoke driver](../tools/smoke/main.go) runs `init` and `onboard` with a preview, a digest-checked write and `check`.
- The [playground smoke](../.github/workflows/playground-smoke.yaml) also runs it in a container.

The dated result below is the record of the retired script.

## Dated result

The published-binary run completed on 2026-10-01 with `v0.5.0` (`windows/amd64`, SHA-256 `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178`). The full machine run took 8.973 seconds; the timed Markitect commands totaled 7.600 seconds. The base and candidate commits were `973f11d9cfc157610ed84f4e9582c3534c67018c` and `26c88c133cf09e09dd7f03a630c8e470128c002d`. The exact durations and passed expected sets are in [`measurement.yaml`](../examples/onboarding/delivery-service/measurement.yaml).

The executable was supplied at a verified local path for this run; package installation itself was setup, not a timed or independently observed step. The first fixture draft also exposed a modeling correction: the policy must be authored in its Text YAML resource and its managed Markdown view regenerated. Context includes the Text YAML input, while impact reports both the changed YAML and its rendered view. The final fixture includes the generated baseline views, and the runner keeps prompts/oracles outside both fixed project snapshots.

These results are one machine exercise and do not establish novice-user usability, general productivity, or provider-client parity. No result should be described as a user study.

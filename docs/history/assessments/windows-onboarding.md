# Windows CLI onboarding exercise

**Retired walkthrough:** the old release downloads and their installation paths were removed on 17 October 2026. This page preserves the original Windows exercise and observations only. Use [current source](../../../README.md#run-the-current-source); original binaries are identified by the [retained release evidence](../distribution-retirement-20261017.md).

This dated exercise measures a machine-run path through a published Windows CLI. It is a reproducible adoption smoke exercise, not novice-user research: one maintainer/agent runs a fixed synthetic task, and the timer does not represent a population or a human learning curve.

## Install and identify the release

The original exercise used the now-retired `v0.5.0` Windows binary with SHA-256 `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178` and version string `Markitect 0.5.0 (windows/amd64)`. Its installation procedure remains in [the pre-retirement README](https://github.com/Glacius-Labs/Markitect/blob/1f4a1ede214d05fd2fee615b3905207df7702426/README.md#install-markitect); the following recorded its installed path, hash and version.

```powershell
$binary = (Get-Command markitect.exe -ErrorAction Stop).Source
Get-FileHash -LiteralPath $binary -Algorithm SHA256
& $binary version
```

The release binary is already installed when the measurement script receives `-MarkitectBinary`; the script records, rather than silently substitutes, its observed version and digest. The `install` command in Markitect is for pinning the CLI distribution into an adopting repository and requires the matching release bundle; this exercise's install step is the documented Windows package installation above.

## Run the bounded task

The synthetic [parcel-support fixture](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/examples/onboarding/delivery-service/README.md) contains shared customer-data guidance, a policy, workflow, contract, agent, and Skill entrypoint. `tasks/query-only-prompt.md` with `tasks/query-only-expected.yaml` is the shared read-only task for MCP/CLI find, explain, and context parity. `tasks/policy-edit-prompt.md` with `tasks/expected.yaml` is a separate CLI-only mutation task that also exercises verification and fixed-SHA impact. Oracle and prompt files are excluded from the measured project's Git snapshots.

The original scripted replay and its later Project/Domain smoke were retired. Their source and procedure remain in [the pre-retirement record](https://github.com/Glacius-Labs/Markitect/blob/1f4a1ede214d05fd2fee615b3905207df7702426/docs/onboarding.md). The current [source smoke](../../../tools/smoke/main.go) exercises canonical Library M1/M2 read-only preparation and refusals; it does not run this historical onboarding or prove Apply.

The dated result below is the record of the retired script.

## Dated result

The published-binary run completed on 2026-10-01 with `v0.5.0` (`windows/amd64`, SHA-256 `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178`). The full machine run took 8.973 seconds; the timed Markitect commands totaled 7.600 seconds. The base and candidate commits were `973f11d9cfc157610ed84f4e9582c3534c67018c` and `26c88c133cf09e09dd7f03a630c8e470128c002d`. The exact durations and passed expected sets are in [`measurement.yaml`](https://github.com/Glacius-Labs/Markitect/blob/d083c8f37b6bb1e09a9c5a7947c421651a7e755b/examples/onboarding/delivery-service/measurement.yaml).

The executable was supplied at a verified local path for this run; package installation itself was setup, not a timed or independently observed step. The first fixture draft also exposed a modeling correction: the policy must be authored in its Text YAML resource and its managed Markdown view regenerated. Context includes the Text YAML input, while impact reports both the changed YAML and its rendered view. The final fixture includes the generated baseline views, and the runner keeps prompts/oracles outside both fixed project snapshots.

These results are one machine exercise and do not establish novice-user usability, general productivity, or provider-client parity. No result should be described as a user study.

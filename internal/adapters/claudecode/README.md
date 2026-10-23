# Claude adapter status

Static `runtimeconfig.AgentProfilePreparer`, closed profile decoding and the pinned Claude wire codec remain available. They establish captured metadata and wire behavior, not native Session qualification.

The retired `run.Agents`/domain Turn engine and assignment consumer have been removed. `Agents` implements the actual `run.Sessions` boundary, but `Open` explicitly returns `run.ErrUnavailable` before reading a credential or starting a container. S2 composition may expose this honest refusal. It must not route a supported-looking Session to the historical engine.

Claude instruction-load priority, normal account-login provisioning and cleanup, background-command/message drain, full hierarchy and actual native tools still need a bounded provider-specific implementation and capability proof. Codex evidence does not qualify Claude.

## Static Runtime input

The closed `claudeCode` section in `markitect.runtime/v1` uses `profileVersion: 1`, an absolute `engineExecutable`, immutable `imageID`, literal `agentdVersion`, ordered captured `instructions`, opaque `credentialRef`, bounded `maxWireBytes`, and explicit `session` configuration. Session fields remain `version`, `model`, `effort`, `systemPrompt`, `builtinTools`, `store`, `cpus`, `memoryBytes`, `pids`, `tmpBytes` and `network`; valid source syntax is not execution support. `RuntimeProfile.Section` only builds explicit configuration bytes and does not start a provider.

Metadata preparation binds the whole captured source, exact image and bounded `/opt/markitect/runtime.env` bytes, recorded Claude `2.1.301 (Claude Code)`, agentd version and selected instructions. `ProfileFromPrepared` refuses incomplete observations, foreign providers/contracts and source or image mismatches. No ambient executable, host login home or credential discovery is used.

The credential shape remains an explicitly selected runtime account token. No credential is resolved on the unavailable Session path. Pure token-redaction and codec tests remain; historical runtime/Turn source tests and evidence are retained on named pre-cutover commits. Do not invoke their old smoke instructions against this replacement source.

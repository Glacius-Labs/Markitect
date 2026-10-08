# Bounded static snapshot trace

Result: the same-version public source handles the unsupported-shell error at the
snapshot-building boundary, but exact candidate behavior and continuation of its
thread initializer remain unproved. No native fatal cause or successful thread
initialization is established. This investigation stops at its research limit.

The sealed candidate remains version `0.162.0-alpha.2`, SHA-256
`3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`,
333357008 bytes. No version/schema/CLI query was repeated. Current local bytes
equal the desktop distribution's bundled CLI bytes. Inspected distribution
metadata identifies Windows package 26.1002.7124.0, Electron 26.1002.52244 and
build 13536; these are distribution identities, not a CLI upstream commit.
The selected package/Owl metadata contains no demonstrated CLI commit or source
attestation. This is a scoped inspection, not a claim about every installed file.

The official version tag resolves through annotated tag object
`08a287137b1e11bf3566d53d6820b82177019f29` to commit
`74e804deeb1241d5fe699b31fb319f7d46454c42`. The tag is unsigned and the release
record reports mutable release metadata. The Windows release EXE is 333357872
bytes with reported SHA-256
`d83cc3582592e307df008411f02f61a93fb93580b53dc173608a63202d97bbe4`.
Its size and digest differ from the candidate. The 864-byte difference is not
explained; signing-only differences or equal executable code are not assumed.
No release binary was downloaded. Tag/version correspondence alone does not bind
the desktop CLI to that source commit.

At that fixed commit, public source supports this conditional control-flow trace:

| Boundary | Exact public source evidence | Consequence in this source |
|---|---|---|
| Unsupported shell | `codex-rs/core/src/shell_snapshot.rs`, lines 539-550: `write_shell_snapshot` rejects PowerShell and Cmd via `bail!` before capture. | The snapshot-writing function returns an error. |
| Immediate creation caller | Same file, lines 285-301: `try_create` awaits the writer, records a warning and maps its error to `write_failed`, then propagates within `try_create`. | This local error is not a successful snapshot. |
| Snapshot builder | Same file, lines 221-244: `build_for_cwd` awaits `try_create`, records success/failure telemetry and uses `snapshot.ok().map(Arc::new)`. | The error becomes `None`, not a propagated `Result` error at this boundary. |
| Public build interface | Same file, lines 151-178: `build` returns `Option<Arc<ShellSnapshotFile>>` and awaits `build_for_cwd`. | Optional absence is exposed to the consuming caller. |
| Session initialization boundary | `codex-rs/core/src/session/mod.rs`, lines 886-930: outer startup awaits `Session::new` and maps a returned initialization error. The nested module is declared at line 258. | This establishes an initialization boundary, but not how `Session::new` consumes the optional snapshot. |

Thus the hypothesis of a handled optional warning is supported at the snapshot
builder in this tag source. The stronger hypothesis that the immediate thread
initializer continues is unresolved: the obtained outer module delegates to a
separate `Session::new` body, which was not obtained within the request budget.
The missing source fact is whether that body tolerates, transforms or otherwise
acts on the absent snapshot, including whether snapshot construction is awaited
or concurrent. This is not filled in from the current latest source or protocol
documentation. Even a complete tag trace would still need a candidate-to-source
binding before being asserted as behavior of the sealed binary.

Consequently there is no basis here to treat the recorded WARN alone as a proved
native initialization failure. Our existing client stderr-terminal rule remains
a sufficient explanation for the recorded client stop; it is unchanged. No
provider-policy, OS/identity, billing, historical R2, quality or S1 inference is
made. The prior private excerpt was not reread or reconstructed; the review uses
only its allowed technical paraphrase and public source material.

Research actually used eight public GET requests and 1006562 response-body bytes
(under 4 MiB), including one 14-byte 404 and one counted repeat of release metadata
after the first response was not retained. Automatic application retries were
not used. Two public source files were retained, plus two curated provenance
records and these findings with one independent review. No repository clone,
binary download, implementation, tests, builds, candidate starts, configuration
changes, purchases or new quota. Historical eight native trees, usage and study
pins remain unchanged. No additional source fetch or follow-on operation occurs.

`local-provenance.json` binds the scoped installed-distribution inspection;
`upstream-provenance.json` binds the tag/release observations, per-request byte
accounting and both archived source digests. Source copies preserve downloaded
bytes and line numbering. Initial unretained responses are identified explicitly;
the provenance records are curated observations, not full HTTP transcripts or a
build attestation.

Public references at the fixed commit:

- [Official tag object](https://api.github.com/repos/openai/codex/git/tags/08a287137b1e11bf3566d53d6820b82177019f29)
- [Official release](https://github.com/openai/codex/releases/tag/rust-v0.162.0-alpha.2)
- [Snapshot source](https://github.com/openai/codex/blob/74e804deeb1241d5fe699b31fb319f7d46454c42/codex-rs/core/src/shell_snapshot.rs#L539)
- [Snapshot error handling](https://github.com/openai/codex/blob/74e804deeb1241d5fe699b31fb319f7d46454c42/codex-rs/core/src/shell_snapshot.rs#L221)
- [Outer session initialization](https://github.com/openai/codex/blob/74e804deeb1241d5fe699b31fb319f7d46454c42/codex-rs/core/src/session/mod.rs#L886)

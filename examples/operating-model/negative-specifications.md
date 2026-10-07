# Negative specifications

These cases define required outcomes for future fresh-agent experiments. They are specifications, not executed proof results.

| Case | Candidate condition | Required outcome |
|---|---|---|
| Ambiguous or missing source policy | The selected .NET Projection has no applicable per-Kind ProjectionPolicy, or two selected policies conflict on the public operation contract. | Escalate before materialization. The Executor must not invent the operation contract or a filename to settle the conflict. |
| Unknown target artifact | An artifact appears under src/Orders/ or src/Billing/ but is absent from the exact selected Projection ownership records. | Keep it UNKNOWN and surface the unresolved path. Do not overwrite or delete it implicitly. |
| Source or policy drift | Any selected Definition, Schema, Module, or ProjectionPolicy byte changes after a request or reviewed plan was created. | Reject the stale request/plan before writing. Rebind against the new exact source revision and review fresh candidate bytes. |

A future run must preserve each negative outcome beside the positive case with its own source, target and invocation bindings. A prepared fixture or passing fixed project check cannot turn any case into an agent result.

# R2 thread-start-only diagnostic: stopped before turn

The single authorized new-runner attempt ended at `server-stderr-activity`. Actual config and requirements gates passed. The client wrote `thread/start` exactly once; no thread response was validated, and actual thread creation remains unknown. Zero turns, task tools, interrupts or Actor-task reservations were requested. No repair or relaunch followed.

The bounded classifier observed 765 stderr bytes, four complete lines, all `stderr.unclassified` with unknown severity/component. Capture was neither capped nor truncated; no raw text was retained. These classifications establish no error cause and do not explain the earlier discarded stderr, even though its recorded byte count was also765. The byte-identical R1 classifier contract retains historical component-membership provenance only; no new-binary component check or emitted-log claim was made.

Source `S`: `c6d84de416986837adfa57967c8b9ef29a1eb90a`. Prestart binding commit `F`: `8b4f0a4d8854dd7d8b7c2fe1863b949c70be2fc9`. Request SHA256 `5780b4a749f766962b09ef4ec89f1ca1b38db20101017440941ed658f0469dbf`; freeze SHA256 `9a189e4baec5024d2941d27020180520c10f411ffb2ae09127f4fa20312a46ed`. New executable SHA256 `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`, version0.162.0-alpha.2 from prior accepted metadata; schema archive SHA256 `3157e8a55329cf4e5346676c3ef0c308402d2420f8f916d6a5b7e0e9c84356ad`. Twelve schema members were repinned; 17 config pairs and thread payload remained unchanged. Seven targeted offline consumer test methods passed in0.004s; old15/31 cases and product suites were not repeated. Independent source/binding reviews preceded the attempt. The binding review corrects one typographical extra letter in the source review's displayed client hash; actual source/request/freeze hashes agree.

| Observation | Actual value |
| --- | ---: |
| New diagnostic reservations / app-server trees | 1 / 1 |
| App-server trees cumulative | 7 |
| New Actor-task reservations / turns / task tools | 0 / 0 / 0 |
| Actor reservations cumulative | 6 |
| New CLI metadata calls / cumulative | 0 / 8 |
| Metadata phase | 0.230711s |
| Native session including cleanup | 0.335774s |
| Cleanup | 0.067306s |
| Controller / limit | 1.407331s / 65s |
| Outer through receipt readback / limit | 2.661528s / 80s |
| Outer tool wall | 2.755615s |
| Native / worker / outer exit codes | 0 / 1 / 1 |
| Discarded stdout / stderr | 50,217 / 765bytes |

Windows Job assignment preceded worker GO; the controller closed the owned Job. Source22 and freeze32 file pins, the two public cwd files and five historical ledgers were checked unchanged after termination. Seven sanitized external receipts were archived byte-for-byte. No raw frames or stderr were archived. The native/worker/outer return codes reflect different layers; native exit0 does not turn the diagnostic into success.

The slot was explicitly released at `2026-10-08T15:58:11.563229+00:00`; Overseer owns the canonical update. Residual authority expired, with no automatic next operation. Historical five model Actor attempts, six Actor reservations, known53,331tokens with aggregate usage unknown, and native counters15/16/13/2250 remain intact. Native internal provider requests/retries, serving model, usage and billing state remain unknown. This attempt does not establish tool capability, OS enforcement, general S1 or comparison quality. S1 remains open and all six study cells remain NOT RUN.

Evidence: [terminal summary](../evidence/s1-thread-start-diagnostic-20261008-r2/terminal-summary.json), [source review](../evidence/s1-thread-start-diagnostic-20261008-r2/preflight-review.md), [binding review](../evidence/s1-thread-start-diagnostic-20261008-r2/binding-review.md), [receipt archive](../evidence/s1-thread-start-diagnostic-20261008-r2/external-archive.json), [postrun verification](../evidence/s1-thread-start-diagnostic-20261008-r2/postrun-validation.json), [postreview](../evidence/s1-thread-start-diagnostic-20261008-r2/postreview.md).

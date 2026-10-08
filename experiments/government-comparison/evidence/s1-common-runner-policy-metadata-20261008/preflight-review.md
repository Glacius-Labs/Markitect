# Independent preflight review: one prospective metadata read

**Disposition: source preflight clear, subject to the separate exact Request/Freeze binding review.** I inspected the final closed client, one-shot wrapper, profile, grant copy, frozen method enums, and recorded offline test result. I ran no tests or process, sent no RPC, and wrote no reservation or ledger entry.

The grant and profile agree on the pinned 0.160.1 executable, public cwd, all 17 config pairs, and the four-message sequence: initialize, initialized, one `config/read`, and one `configRequirements/read`. The only permitted inbound notification is the schema-validated disabled remote-control status. Warnings, provisional content, stderr activity, other notifications, malformed or unexpected frames, missing/ambiguous required values, and resource/deadline failures stop without retry. The new client preserves missing/null/present/invalid states, only retains allowlisted values and opaque custom-profile aliases, and persists no raw frames, configuration, or stderr.

The prelaunch path binds the reviewed plan, profile, grant, source thread, live grant and active slot, source commit, executable/interpreter and schema pins, five historical ledger hashes, request/freeze/reservation, and the new cwd contents. The client requires the one-time reservation and clean frozen commit before controller startup. A gated worker is assigned to the existing Windows Job before it receives `GO`; the worker checks the job-assignment receipt against its PID, parent PID, and binding before launching the target. Exclusive markers prevent a second controller, worker, or target start. The shared in-flight reservation accounts for concurrent stdout/stderr reads against the single 4,000,000-byte ceiling; stderr bytes are queued immediately and cause a stop. No raw process logs are captured by the outer controller.

The timing and volume bounds match the grant: 33 seconds active RPC work, 38 seconds for the owned process, up to 10 seconds for cleanup within the 50-second controller bound, and 60 seconds through outer receipt readback; queue capacity is 32 frames and notifications are limited to 16. These are enforced watchdog/receipt limits, not a hard real-time guarantee. The frozen plan’s interpretation is retained: success can establish only observed argv acceptance and a reported prospective configuration precondition. It cannot prove that the inline key is recognized or enforced, active thread permissions, OS/tool access, file-read success, serving model, provider request/turn counts, or the historical denial cause.

The recorded offline gate reports 13/13 focused tests passing and zero Codex processes, metadata requests, model calls, or live-ledger writes. The tests cover the parser/classifier/assessment/profile/cwd/live-authority paths and mocked Job-before-GO success/failure. They do not establish real Job containment, app-server behavior, effective permissions, or provider/tool behavior. The earlier read-budget race and missing live slot-status check were corrected in the reviewed source; the final source has shared in-flight byte reservations and exact live grant/slot status checks.

Reviewed hashes:

- `client.py`: `8b6a2032dce5468c730885906f030fcfbed3cb4f23b010e95f90592a38b0fdfb`
- `run_once.py`: `d8345a03dd04186a769e5ed4d0c4e711892b3007078fd598d3b2ab3aeaa7e6e3`
- `profile.json`: `de8de70e51b66ca4b7d3bf8a9c01ec577c462e258b824926ea9e6f85c1742589`
- `authorization-grant.json`: `1381f180e45d90399e605af5556cf040c1aeca77885c6608faa095b605c6a853`
- `frozen-method-enums.json`: `6553df9ac4a37d11402728992ed5e684ea379fbadeee790700867ae4542594cf`
- `test_client.py`: `a30c7bf6661df1e92952618cd169584f7c68e7f2ff0a0188e32fbef0d967e44b`
- Offline validation receipt: `b184eae5d778ea57ac6d892ab667ea4849ec72cef3bee4490dc7bc36c3d02030` (13 focused tests; no process or ledger writes)

This review does not authorize the start. Before that, the final committed source and exact generated Request/Freeze artifacts require a separate read-only binding review. Any mismatch or missing precondition remains a terminal stop.

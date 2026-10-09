# Independent variant validation

Status: implementation source `d360ec0199d4333271fd6ec6c66227a2da4aca83` is frozen and pushed. Its full fixed-source Verify FAILED at two projectcli tests. Both bounded fixture repairs have passed their targeted reruns and parent review; a fresh fixed-source full replay is next. Accepted-Main merge remains pending. Source labels below distinguish coordinated working inputs from commits. No release, real provider proof, case study or human acceptance is asserted.

## Focused checks

KG04 ProjectModel tests and vet passed, and an independent Luna High reader rechecked the two repaired findings: private foreign Decision subjects and missing Impact routing to a referenced subject owner. Pure bidirectional traversal tests and vet passed, including orientation, cycles, deterministic witnesses and truncation. Parent architecture gate, authoring timeout boundary/schema tests, shared service packages and whole-repository vet passed on coordinated working inputs. These are component results, not a full Verify.

The protocol transport's initial full projectcli test attempt was interrupted after more than two minutes of silence, without a reported test failure. It is INTERRUPTED, neither PASS nor FAIL. The focused protocol/CLI tests passed; parent will run the fixed candidate with the required generous suite limit. The earlier full failed source Verify at 08b273f6 and its exact original log remain unchanged.

## Actual provider-free CLI journey

An isolated Git-root copy of examples/project-knowledge was initialized on kg-user-journey. Initial development attempts found the nested Git-root guard, forbidden multiple YAML documents in one file, and unclassified README full coverage. The example now has separate selected Decision files and an explicit README Artifact; the guard remains unchanged. The corrected copy passed project check with accounted/conforming full repository coverage. All six knowledge actions passed on fixed fixture revision 6f0931abf295723fdff09c00ff7c99607636c49d, including an explicit bidirectional Statement/Artifact/Check witness and IdentityChange history. The binary was built from coordinated working source; its SHA256 was bca9c5a8e34fafa8b061643e2ddd0fea769c5689ce22fb1e90afb44f1d57aa7c. Later review fixes require a final source-bound replay before handoff. No model calls occurred. This tests transport/compiler mechanics with declared fixture data, not autonomous engineering quality.

## Independent Host review

The reader identified leaked peer Manager IDs in a shared Exploration scope, a hidden-inventory coverage side channel, unsupported review/verification/history graph kinds and edges, observed-looking links for absent runs/tasks, and missing child Manager Purpose. Assigned owners closed those findings with targeted regressions and a bounded reread. Parent additionally repaired missing nearest-Manager ownership edges, unjoined history records, strict duplicate/case-alias JSON decoding, and a public example coverage gap before freezing the implementation source.

## Timing and configuration

The source go-tests check and explicit Go test-binary limit are 60 minutes; hosted quality outer limit is 90 minutes. The Host authoring ceiling is coherently raised to 3600 seconds, boundary tests and generated Project schema updated, and the exact pipeline bytes repinned. This does not change actor/study budgets or preserve a failed run as successful. Focused checks use 10 minutes, package checks 30 minutes. The original failed Verify is immutable.


All five Host findings and the raw ExtraFacts visibility residual are closed by the independent reader. Parent service packages passed after the fixes; scope service targeted tests/vet and the final hidden-ID spotcheck passed. The protocol now rejects duplicate keys recursively and typed field case aliases. Model, file and definition ownership remain distinct; direct public contract owner nodes carry identity only. History has explicit current-definition and event/resolution/briefing links, with no invented removed live node. Whole-repository artifact and module checks pass on the accounted working candidate.

## Final source-bound actual binary journey

The final journey replay passed on implementation source `d360ec0199d4333271fd6ec6c66227a2da4aca83`, replacing the earlier working-source journey as the source-bound transport evidence. An isolated Git-root example fixture used revision `963a5f1e00ada83eb59072f60f426c793117d034`. Binary SHA256: `ac709206ef05e14780e423cca899d154a676ac5ae2667f25bc755032206ae24a`. Project check returned exit 0 and accounted/conforming coverage. All six fixed Manager-scoped actions returned exit 0, with one stable graph digest; the trace included a bidirectional path and history included IdentityChange. An actual stdio binary process passed initialize/initialized, listed six tools, and returned the same graph digest through knowledge_graph, exit 0 and no tool error. No provider calls occurred.

The exact [JSON receipt](evidence/user-journey-d360ec01.json) has SHA256 `84d2a85899a0dfd65fe0197b24ddc1ef4eec67311ec8abbd54a42f231c91f00a`. Its graph digest is `sha256:d7d390d24a653dd709fee56cb60a48aa0f0704ff7dac17bd7db29c22abd43b44`. This verifies ordinary compiler/query/transport mechanics with declared fixture data; it does not prove real actor quality or human acceptance.

Complete run/candidate/check/review/verification/Apply chains are tested in the validated ProjectRun port fixtures. The evidence-to-graph real-loader fixtures cover Exploration, Brownfield and a planned run with missing execution; they do not constitute a fresh complete operational chain end to end. Hosted CI and current-provider runtime fingerprint comparison remain unestablished. See the [handoff](continuation-handoff.md) for the remaining accepted-Main integration and sandbox evaluation.

## Fixed-source Verify at d360ec01

Actual terminal result: FAILED, process exit 1, 1,748,072 ms overall. The Go suite ran 1,529,508 ms and exited 1 within its 3,600,000 ms limit; this was not a timeout or interrupted run. Architecture/import, managed-artifact and module checks exited 0. All packages other than projectcli passed, including ProjectModel, projectknowledge, projectgraph, knowledgeevidence, knowledgeprotocol, projectapp and projectrun. Claude/Codex protocol checks were NOT RUN because Verify stopped at go-tests.

The two reported failures were TestBrownfieldStagedManagerLoopBeginContextProposeAndIntegrate (a proposed child Manager had a null instead of an explicit empty DelegationEvidenceIDs list), and TestKnowledgeCLIAndMCPShareTheSameProjectService (its old private-node negative targeted a now-intentionally-visible identity-only owner of a direct public contract). Repairs must preserve the validator and authoritative visibility rules, add a genuinely private target negative, and assert that the permitted owner node contains no private fields.

The [raw log](evidence/verify-d360ec01.log) has SHA256 `08d096b27385fc97c2ba61053e573758196c54ff3e10c1c086b3863c297b129c`; the [execution receipt](evidence/verify-d360ec01-receipt.json) records actual start/end and exit. Source snapshot: `af2d334d8c2e3d4c3dbbe0cc3ac66a3b579845165ae1e6296f8c72d41b99435a`. Tool digest: `sha256:b14266bc8f8f339e37748aa8e3b960f23c393fbd29f7b9b34c7b41a40f573e16`. This failed result and the older 08b273f6 log remain immutable; a later result is separate evidence.

## Bounded post-Verify fixture repairs

Luna High fixture owners changed only projectcli tests; production model/graph/evidence/visibility/transport behavior is unchanged from d360ec01. The staged Brownfield proposal explicitly supplies an empty child DelegationEvidenceIDs list, preserving its root-to-child evidence assignment and no further child delegation. Its exact test passed in 42.376 s with count=1 and a ten-minute limit.

The shared Knowledge CLI/MCP test keeps whole-project transport parity and adds an actual Orders Manager CLI graph for the identity-only public-contract owner assertion. Its forbidden target is now an Inventory-owned private Artifact. The private response is decoded, requires isError=true and absent structuredContent, matches the exact generic error text, and excludes the JSON-encoded private target ID. Parent caught and corrected the initially mistaken whole-project basis of the new field assertion and the raw-string escaping weakness before freeze. `go test ./internal/host/projectcli -run Knowledge -count=1 -timeout=10m` passed. No validator or visibility rule was weakened. These focused results do not relabel the d360ec01 full Verify as passing.

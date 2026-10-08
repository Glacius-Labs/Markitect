# Independent review: `government-scope-identity-offline-20261008`

**Verdict: PASS; no material findings remain.** Reviewed the source diff from clean base `1ea3fc970797e01206e98d15e1951021b4ee5586` against pinned Government source `04e225d5caee78c2a198607143863fca1e829750`. All six retained native contract files match their original Git objects byte-for-byte. The R4 report was used only as a read-only format example; this review does not translate, replay, or rescore R4.

The pinned `core.DefinitionIdentity.Key()` encodes `[apiVersion, kind, namespace, name]` as JSON. The adapter now decodes that representation and compares all four original strings without trimming, case folding, delimiter joining, or a namespace/name fallback. Identity validation matches the pinned core grammar, including an empty namespace. Review coverage still requires every planned integration review to appear in a passing review actor's scopes; malformed, partial, invalid, or differently identified scopes fail closed. Queue and Resume translation both reach the same run-report validator.

Adjacent cabinet and vote checks now compare complete four-field identities. This closes the same concrete namespace/name-only equality defect. Candidate, evidence, round, receipt, role, selected-root-review, final-vote, decision, and promotion checks remain in place. The short namespace/name vote label remains presentation data; it is not used for identity authorization or equality.

The recorded focused run reports **29 tests passed** across `test_government` and `test_government_scope_identity`; the harness record states `subprocess.Popen` was denied. The new cases distinguish valid native JSON scopes, escaped/whitespace-formatted encodings of the same identity, invalid matching identities, incomplete/malformed scopes, field mismatches, missing review coverage, and cabinet/vote identity mismatches. No product, native, controller, wrapper, delegate, Actor, model/provider, metadata, Go, or full-suite execution was part of this review. No human or semantic acceptance is implied.

## Reviewed inputs (SHA-256)

Source and test files:

- `runtime/government.py` — `531c4819d5a258280478d4b5f808602a326ed7e2eea157200e00d6ddd56e9fb1`
- `runtime/test_government.py` — `f63a85036435c6c0fb63421a84330c86b48ceb911988aa2167b498d214b96f49`
- `runtime/test_government_scope_identity.py` — `c390822efbad3f82750bf42b99e7c34ad323e0e9831d1b9caa9dd4d16cc6e33c`

Pinned native contract snapshots (each independently matched to the original Git object at the pinned commit):

- `pinned-contract/internal/core/types.go` — `04f1f3ebd8b6cff5c22af549fbe1e3d1451b296df4afa12613e2cedf3a24410f`
- `pinned-contract/internal/core/compile.go` — `849f797e50a4c69758a87b43fa1fff2575d75ee0f9ed3fe4a5764c0db6b8096e`
- `pinned-contract/internal/host/government/execution/run.go` — `4bf98260f508108aed6e7e01639d336fc1553fac5d3fee641588589a582d68f1`
- `pinned-contract/internal/host/government/execution/actors.go` — `f56851a1c72bb8a4668d6dc723509f1f99503d5af36d557dbf42cad7162422b7`
- `pinned-contract/internal/host/government/plan.go` — `9a3231bbad502f837ac28a2ac90cbc9b522d9084f6f3157031ec1831f0466088`
- `pinned-contract/internal/host/government/bindings.go` — `3435ee5a95080237f47f0b3dcd8b8dce6b45c176f86aec66baa7f0aab32c7f92`

Additional evidence inputs:

- `native-contract-example.json` — `88cf9b58e82e38aaba3ad72bdf75c63aee0e34f25df21d88277d4f9b857ad745`
- `focused-tests-2.log` — `ae4dab924ff54b3a5c63cf088aacfbd7afe393c1a84496f548a8603459d26a60`

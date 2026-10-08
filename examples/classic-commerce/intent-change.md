# From owner intent to a later rule change

This public source walkthrough uses the published v0.14.1 experimental canonical alpha. It is a deterministic development example, not an agent-quality trial or a new release. Start with the [example prerequisites](README.md#prerequisites). The Core compiles supplied structure; it does not translate conversation into business semantics.

## Decide what applies

A concrete starting request is: “Create an order total from quantity and unit price. Quantity must be positive, price non-negative, and invalid inputs must be rejected. Keep the application boundary and publish readable documentation.”

An author proposes the canonical Definitions and selected projection policies. The owner reviews their meaning, boundaries and omissions before accepting that source revision. A Git commit fixes the reviewed bytes; it does not authenticate who accepted them. Keep alternatives and conversation outside the canonical model. Nothing here requires a separate permissions model for normal implementation choices.

| Accepted intent or contract | Canonical owner / explicit input | Representation or independent evidence |
|---|---|---|
| Quantity, price, total and invalid inputs | `definitions/create-order.use-case.yaml`, `purpose` | `src/Commerce/CreateOrderHandler.cs`; finite cases in `checks/Program.cs` |
| Handler public signature | `definitions/create-order.projection-policy.yaml` | `CreateOrderHandler.Handle(int, decimal)` |
| Application boundary | `definitions/create-order.effect-axis.yaml`, `spec.boundary`, and `definitions/effect-axis.projection-policy.yaml` | `src/Commerce/EffectAxis.cs`; separate boundary assertion |
| Selected .NET representation | `definitions/commerce.projection.yaml` plus `canonical.yaml` Module binding | Three declared files under `src/Commerce/` |
| Readable representation | `definitions/commerce.markdown-projection.yaml` and selected Markdown policies | `docs/represented/index.md` |
| Build and finite business checks | `canonical.yaml` named check; five runtime `checkInputs` | Immutable check receipts for local and parent assurance scopes |

Paths in the table are relative to this example unless they begin with `src/` or `docs/represented/`, which are disposable repository outputs. The independent probe repeats selected business expectations intentionally; maintaining it when intent changes is visible work, not something Core infers. Passing these cases does not cover every possible rule or input.

Inspect a committed model before implementation. In the disposable repository prepared by the runner below, these commands use the same public CLI:

```powershell
$source = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot bind accepted source' }
& $binary canonical --repo . --config examples/classic-commerce/canonical.yaml --action model --revision $source
if ($LASTEXITCODE -ne 0) { throw 'Model is invalid' }
& $binary canonical --repo . --config examples/classic-commerce/canonical.yaml --action context --revision $source --api-version commerce.example.org/v1 --kind UseCase --namespace commerce --name create-order
if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect selected intent' }
```

Review the selected Definition, its explicit references, policies, check inputs and intended generated targets. Structural validity does not decide whether this is the right business requirement. For another project, author its own vocabulary, checks and bindings; do not reuse this example's fixed actor as an intent translator.

## Change one accepted rule

Later the owner asks: “Require at least two items per order.” The proposed source change replaces the UseCase purpose with:

```yaml
purpose: >
  Creates an order total from an integer quantity of at least two and a
  non-negative decimal unit price. Returns quantity multiplied by unit price.
  Rejects quantity below two and negative unit price with ArgumentOutOfRangeException.
```

The independently authored check changes the explicit quantity-one success case to rejection. Its negative-price case uses quantity two so that the new quantity guard cannot conceal a missing price guard. The handler signature, decimal multiplication and application boundary remain applicable without new permissions or policies.

The .NET policy owns the representation signature and refers to the selected UseCase for business behavior; it does not maintain a second minimum-quantity rule.

The new source commit is distinct from the original accepted source and the immutable post-Apply evidence revision. Inspect `impact --base OLD --revision NEW` and `reconcile-plan`; do not edit the emitted Markdown as a second authority. Both representations must reflect the changed intent, while `EffectAxis.cs` and the project file retain their valid bytes. Old verification remains historical. An incomplete audit and refused old Apply are expected before current work is materialized and verified.

## Run the public mechanical example

The [example README](README.md) links the copyable command and finite smoke profile. The runner creates a separate feature-branch repository from exact Git blobs, records the proposed canonical/check diff, and requires explicit selection of this known change. It exercises the public native commands and records each source revision, reviewed output digest, raw command result and check receipt outside the source checkout. It never changes your original checkout or published binary.

The two fixture selections authorize only the documented known source change and exact candidate-byte review in that disposable repository. They are not authenticated human acceptance, a general agent mandate or proof that arbitrary generated code is correct. A real workflow substitutes the project's ordinary owner review and an independently reviewed candidate before guarded Apply.

A failed prerequisite, stale binding, failed check or missing evidence stops the relevant cycle visibly. Preserve the complete attempt and its cumulative usage. The driver does not install tools, invoke a model/provider, retry automatically, publish or replace a study candidate.

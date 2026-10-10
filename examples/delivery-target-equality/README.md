# Delivery target equality pressure fixture

This technology-neutral Delivery model records three independent facts: each `Service` belongs to one `Product`, each `Environment` belongs to one `Product`, and each `Deployment` names one `Service` and one `Environment`. Its Domain policy compares the fixed paths `Deployment → Service → Product` and `Deployment → Environment → Product`; it does not declare a general graph query.

The fixture contains two passing Deployments: `orders-production` resolves both paths to `commerce`, and the independent `telemetry-production` resolves both to `telemetry`. Tests privately change either owner reference on the orders path to `support-tools`. The references remain well-typed and resolvable, but the Deployment receives one failed per-resource PolicyResult with both endpoints and path steps in the comparison trace. The unrelated telemetry Deployment remains passing.

The pre-operator regression test starts from a private snapshot, removes only the new equality constraint, then applies a typed owner mismatch. It confirms the rest of the finite model language accepts that graph with no diagnostics or policy results. This compares the language before and after this one operator; it does not claim that a released Markitect version shipped this exact fixture.

The Service and Environment Product references remain separate canonical ownership facts. If a project can derive one fact without losing meaning, it can normalize its model first. A project-owned check or configured adapter is also suitable when the equality rule is specific to that adopting project. This example does not establish a real business risk or claim that every project should adopt the rule. It is separate from the Release / ReleasePolicy Delivery example.

Both Product ownership relations deliberately have `context: false` and `invalidate: false`. A Deployment review Skill receives the Deployment, its Service, and Environment, but not their Products solely through those ownership relations. The policy engine still records the two path dependencies: changing either Product owner affects the orders Deployment. The unrelated telemetry cohort stays outside that policy impact.

The executable tests are `TestDeliveryTargetEqualityCurrentLanguageAcceptsMatchingAndMismatchingDeployments`, `TestDeliveryTargetEqualityPreOperatorKernelAcceptsTypedMismatch`, `TestDeliveryTargetEqualityReportsDifferentPathResults`, and `TestDeliveryTargetEqualityOwnershipIsNonContextAndInvalidatesPathDependents`.

From the repository root:

```powershell
go run ./src/cmd/markitect-legacy format --repo examples/delivery-target-equality
go run ./src/cmd/markitect-legacy check --repo examples/delivery-target-equality
go run ./src/cmd/markitect-legacy model --repo examples/delivery-target-equality
go run ./src/cmd/markitect-legacy context --repo examples/delivery-target-equality --namespace engineering --kind Skill --name deployment-review
go test ./src/harness/examples -run DeliveryTargetEquality -count=1
```

# Canonical projection

## fulfill-order-handler

Kind: Handler (commerce.example.org/v1)  
Namespace: Fulfillment

Purpose: Implements fulfillment while preserving the public API and shared quantity obligation.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: Identifies a public operation implementation boundary.

### Properties

- `operation` (reference, 1..1): UseCase realized by this Handler.

### Values

~~~json
{
  "operation": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "UseCase",
    "name": "fulfill-order",
    "namespace": "Fulfillment"
  }
}
~~~

## create-order-handler

Kind: Handler (commerce.example.org/v1)  
Namespace: Orders

Purpose: Implements order creation while preserving the public API and shared quantity obligation.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: Identifies a public operation implementation boundary.

### Properties

- `operation` (reference, 1..1): UseCase realized by this Handler.

### Values

~~~json
{
  "operation": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "UseCase",
    "name": "create-order",
    "namespace": "Orders"
  }
}
~~~

## effect-boundaries

Kind: Rule (commerce.example.org/v1)  
Namespace: shared

Purpose: Defines behavior around visible effects and independent business-area boundaries.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: States a shared business obligation or broad operating rule.

### Properties

- `maximum` (integer, 0..1): Inclusive upper quantity boundary.
- `minimum` (integer, 0..1): Inclusive lower quantity boundary.
- `statement` (string, 1..1): Canonical rule text.

### Values

~~~json
{
  "statement": "Validate applicable inputs before invoking callbacks. Rejected input returns false without invoking a callback. For accepted input, invoke the callback exactly once; if it returns normally, return true. For any callback exception, do not retry. If the callback throws InvalidOperationException or a subclass, return false; propagate every other exception as the same exception instance. Callback effects may have occurred before an exception; no rollback is promised. Preserve area boundaries and add no hidden side effects or shared mutable state."
}
~~~

## shared-quantity-range

Kind: Rule (commerce.example.org/v1)  
Namespace: shared

Purpose: Defines the common valid quantity range for Orders and Fulfillment.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: States a shared business obligation or broad operating rule.

### Properties

- `maximum` (integer, 0..1): Inclusive upper quantity boundary.
- `minimum` (integer, 0..1): Inclusive lower quantity boundary.
- `statement` (string, 1..1): Canonical rule text.

### Values

~~~json
{
  "maximum": 10,
  "minimum": 1,
  "statement": "A requested quantity must be a whole number from 1 through 10 inclusive."
}
~~~

## fulfill-order

Kind: UseCase (commerce.example.org/v1)  
Namespace: Fulfillment

Purpose: Fulfills an order quantity under the same shared quantity Rule used by Orders.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: Describes one externally meaningful bounded-area operation.

### Properties

- `boundary` (string, 1..1): Callback, validation, and composition behavior.
- `handledBy` (reference, 1..1): Handler realizing this operation.
- `nextUseCase` (reference, 0..1): Next composed operation, if any.
- `publicApi` (string, 1..1): Exact public C# entry point.
- `quantityRule` (reference, 1..1): Shared quantity Rule.

### Values

~~~json
{
  "boundary": "For a nonnull dispatch callback, reject invalid quantity before dispatch; for valid input, invoke dispatch exactly once with the original quantity. Callback result and exception semantics are defined by the shared effect-boundaries Rule. Neither library may reference the sibling project. Caller composition is external; no third generated orchestrator is required.",
  "handledBy": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Handler",
    "name": "fulfill-order-handler",
    "namespace": "Fulfillment"
  },
  "publicApi": "namespace Fulfillment; public sealed FulfillOrderHandler(Action\u003cint\u003e dispatch); public bool Execute(int quantity).",
  "quantityRule": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Rule",
    "name": "shared-quantity-range",
    "namespace": "shared"
  }
}
~~~

## create-order

Kind: UseCase (commerce.example.org/v1)  
Namespace: Orders

Purpose: Creates an order when the SKU and requested quantity satisfy their declared input Rules.

Schema purpose: Quantity-bounded operations with explicit cross-area references.
Kind purpose: Describes one externally meaningful bounded-area operation.

### Properties

- `boundary` (string, 1..1): Callback, validation, and composition behavior.
- `handledBy` (reference, 1..1): Handler realizing this operation.
- `nextUseCase` (reference, 0..1): Next composed operation, if any.
- `publicApi` (string, 1..1): Exact public C# entry point.
- `quantityRule` (reference, 1..1): Shared quantity Rule.

### Values

~~~json
{
  "boundary": "For a nonnull persist callback, reject invalid quantity or a SKU that is null, empty, or consists only of .NET whitespace characters before invoking the callback. A SKU containing any non-whitespace character is valid under the SKU rule even with leading, internal, or trailing whitespace; preserve it exactly. For valid input, invoke persist once with the original SKU and quantity. Callback result and exception semantics are defined by the shared effect-boundaries Rule. Neither library may reference the sibling project. Caller composition is external; no third generated orchestrator is required. The external Orders callback calls Fulfillment.Execute once with the original quantity. If Fulfillment returns false, the adapter throws InvalidOperationException; Orders maps that callback failure to false. When the Fulfillment dispatch callback throws InvalidOperationException, this failure path invokes dispatch once and the Orders adapter once, returns false from Orders, and lets no InvalidOperationException escape. Normal callbacks preserve successful composition; no rollback is promised.",
  "handledBy": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Handler",
    "name": "create-order-handler",
    "namespace": "Orders"
  },
  "nextUseCase": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "UseCase",
    "name": "fulfill-order",
    "namespace": "Fulfillment"
  },
  "publicApi": "namespace Orders; public sealed CreateOrderHandler(Action\u003cstring,int\u003e persist); public bool Execute(string sku, int quantity).",
  "quantityRule": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Rule",
    "name": "shared-quantity-range",
    "namespace": "shared"
  }
}
~~~

## Projection policies

### handler-markdown

Renders handler ownership in both business areas.

~~~json
{
  "guidance": "Preserve handler identity, purpose, and typed UseCase reference.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Handler"
  },
  "targetTechnology": "markdown"
}
~~~

### rule-markdown

Renders shared Rules and their source identities.

~~~json
{
  "guidance": "Preserve Rule identity, statement, and typed bounds accurately without adding implementation semantics.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Rule"
  },
  "targetTechnology": "markdown"
}
~~~

### usecase-markdown

Renders operation public APIs, boundaries, references, and composition.

~~~json
{
  "guidance": "Preserve operation identity, publicApi, boundary, and typed references without claiming adoption.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "UseCase"
  },
  "targetTechnology": "markdown"
}
~~~

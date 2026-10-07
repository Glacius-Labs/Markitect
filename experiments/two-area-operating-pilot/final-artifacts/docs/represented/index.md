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
  "statement": "Validate before visible effects; preserve clear business-area boundaries; introduce no hidden side effects or shared mutable state."
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
  "boundary": "For a nonnull dispatch callback, valid input returns true and calls dispatch exactly once with the original quantity; invalid input returns false and calls it zero times. Neither library may reference the sibling project. Caller composition is external; no third generated orchestrator is required.",
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

Purpose: Creates an order when the requested quantity satisfies the shared quantity Rule.

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
  "boundary": "For a nonnull persist callback, valid input returns true and calls persist exactly once with the original quantity; invalid input returns false and calls it zero times. Neither library may reference the sibling project. Caller composition is external; no third generated orchestrator is required. When a caller wires the Orders callback to Fulfillment.Execute, valid quantities reach the Fulfillment dispatch callback exactly once and unchanged; rejected orders trigger no downstream call.",
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
  "publicApi": "namespace Orders; public sealed CreateOrderHandler(Action\u003cint\u003e persist); public bool Execute(int quantity).",
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

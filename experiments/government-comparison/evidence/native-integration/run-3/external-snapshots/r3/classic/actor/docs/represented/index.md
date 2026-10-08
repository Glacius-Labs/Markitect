# Canonical projection

## create-order-effects

Kind: EffectAxis (commerce.example.org/v1)  
Namespace: commerce

Purpose: Keeps externally visible order effects behind the application boundary.

Schema purpose: Independent example application vocabulary used to demonstrate ontology-neutral Projection binding.
Kind purpose: Declares an application operation's effect boundary and ordering intent.

### Properties

- `boundary` (string, 1..1): States the explicit application boundary for this effect.

### Values

~~~json
{
  "boundary": "application"
}
~~~

## create-order-handler

Kind: Handler (commerce.example.org/v1)  
Namespace: commerce

Purpose: Coordinates the application behavior for creating an order.

Schema purpose: Independent example application vocabulary used to demonstrate ontology-neutral Projection binding.
Kind purpose: Describes one application component responsible for handling a use case.

## create-order

Kind: UseCase (commerce.example.org/v1)  
Namespace: commerce

Purpose: Allows a customer to create an order.

Schema purpose: Independent example application vocabulary used to demonstrate ontology-neutral Projection binding.
Kind purpose: Describes one independently meaningful unit of application behavior.

### Properties

- `handledBy` (reference, 1..1): Names the handler realizing this use case.

### Values

~~~json
{
  "handledBy": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Handler",
    "name": "create-order-handler",
    "namespace": "commerce"
  }
}
~~~

## Projection policies

### effectaxis-markdown

Document the selected canonical meaning with explicit source provenance.

~~~json
{
  "guidance": "Preserve canonical purpose and declared relationships; generated prose has no canonical authority.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "EffectAxis"
  },
  "targetTechnology": "markdown"
}
~~~

### handler-markdown

Document the selected canonical meaning with explicit source provenance.

~~~json
{
  "guidance": "Preserve canonical purpose and declared relationships; generated prose has no canonical authority.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "Handler"
  },
  "targetTechnology": "markdown"
}
~~~

### usecase-markdown

Document the selected canonical meaning with explicit source provenance.

~~~json
{
  "guidance": "Preserve canonical purpose and declared relationships; generated prose has no canonical authority.",
  "sourceKind": {
    "apiVersion": "commerce.example.org/v1",
    "kind": "UseCase"
  },
  "targetTechnology": "markdown"
}
~~~

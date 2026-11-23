# Capability

What the product can do for its users in one field, such as the lending or the catalog of a library. Take it for each part of the product, as the README of its folder, with the documents it links beside it; a large capability has parts, each a folder with its own README of this type. Cut capabilities by what belongs together for the users, not by the building blocks of the code: one may run across several, and splitting a building block never moves it. Take the [domain](domain.md) instead to describe what the system manages from the inside, and an [operation](operation.md) for one thing a caller can do.

## For
The [stakeholders](stakeholder.md) it serves, as links, one per line. Example: `[Members](../members.md)`.

## Offers
What its users can do with it, as links to [operations](operation.md), one per line. Example: `[Renew a loan](renew-a-loan.md)`.

## Covers
The [concepts](../core/concept.md) and [rules](../core/rule.md) it holds, as links, one per line. Example: `[Loan](loan.md)`.

## Parts (optional)
The capabilities below it, each as a link to its README. Example: `[Reservations](reservations/README.md)`.

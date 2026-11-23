# Context

What is given from outside and cannot be changed by the project: the words of the field, the systems it works with, the limits and the assumptions. Take it once per system, as the README of its context part. Take the [domain](domain.md) instead for what the system itself manages and does.

## Terms
The words of the field, as links to [terms](../core/term.md), one per line. Example: `[Due date](due-date.md)`.

## External systems
The [external systems](external-system.md) it works with, as links. Example: `[Town payment service](town-payment-service.md)`.

## Constraints (optional)
Laws, standards and other limits, each as a link to the [rule](../core/rule.md) that imposes it. Example: `[Reader data is kept for two years at most](rules/reader-data.md)`.

## Assumptions (optional)
What the system takes for granted so that it can be simpler: for each assumption, who ensures it and what is therefore not built. Example: `Every member has one card number, issued by the town; the town ensures it is unique, so the system does not check duplicates.`

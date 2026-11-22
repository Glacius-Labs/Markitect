# Architecture

How the system is built: its approach, its building blocks, the decisions behind them and where it runs. Take it once per system, as the README of its architecture part. Take a [component](component.md) for one building block, and the [domain](domain.md) for what the system does rather than how.

## Strategy
The fundamental approach, in a few sentences. Example: `One program with a database; the desk and the member page use the same operations.`

## Components
The [components](component.md) it is built from, as links, one per line. Example: `[Loans](components/loans.md)`.

## Decisions
The architecture [decisions](../core/decision.md), as links, one per line. Example: `[Loans are kept per copy](decisions/loans-per-copy.md)`.

## Deployment
Where and how the system runs. Example: `One server in the library office; a nightly backup to the town cloud.`

## Rules (optional)
The [rules](../core/rule.md) that hold across the code, as links. Example: `[Only the loans component writes loans](rules/loans-written-once.md)`.

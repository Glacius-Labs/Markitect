# Component

A part the system is built from, such as a module, a service or a library of its own. Take it for each building block that has a responsibility of its own and that other parts rely on. Take an [operation](operation.md) instead for one thing a caller can do, and the [architecture](architecture.md) for how the blocks fit together.

## Responsible for
What it does, with links to the concepts and operations it realizes. Example: `Keeps the loans: realizes [Lend a copy](../domain/lend-a-copy.md) and the concept [Loan](../domain/loan.md).`

## Uses (optional)
Other components and external systems it uses, as links. Example: `[Catalog](catalog.md), [Mail service](../context/mail-service.md)`.

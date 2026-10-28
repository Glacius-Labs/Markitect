---
kind: std/Project
---
# Library

One small library manages registered [[members]], physical [[catalog|book copies]] and [[lending|loans]]. A trusted operator registers members and copies, changes membership status, borrows and returns copies, inspects current state and requests [[lending/reminders|overdue notices]].

A member is domain data, not a login account. A new library starts empty; registration is ordinary supported behavior, not hardcoded example data. Operations are processed sequentially against current state. Accepted changes remain available to later operations in that state; persistence across process restarts is outside this example.

There are no reservations, renewals, fees, lost or damaged books, branches, authentication, automatic scheduling or message delivery. Records are not edited or deleted except for member status and loan return state.

[[inputs]] governs all operations and [[realization]] governs the delivered implementation, tests and documentation. The [[callable-interface]] fixes the program and its wire format.

# Product

What the product can do for its users, by capabilities. This part has no fixed chapters: its capabilities are the product's own, and the user [handbook](handbook.md) is built from it.

Fill it capability by capability. A capability is a folder with a README of type [Capability](../../packages/software/capability.md): whom it serves, the [operations](../../packages/software/operation.md) it offers and the [concepts](../../packages/core/concept.md) and [rules](../../packages/core/rule.md) it covers, each a document of its own beside the README. A large capability has parts, each a folder with its own README, linked from the capability's README. Structure it, never flat: when a folder holds many documents side by side, they belong into capabilities. Cut the capabilities by what belongs together for the users, not by the building blocks of the code.

Example for a lending library: `lending/` with `README.md`, `loan.md` and `renew-a-loan.md`; `catalog/`; `members/`. What lasts of a feature of the plan reaches its capability through the Target changes of the feature's stories, which land with their realization.

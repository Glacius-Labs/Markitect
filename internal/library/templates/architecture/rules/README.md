# Rules for building

The rules that hold across the code: how the building blocks may depend on each other, and what no block ever does. One [rule](../../../packages/core/rule.md) per document, whose Applies to links the building blocks it governs; link each from the section Rules of the architecture's README.

Write a rule where a builder could get it wrong, such as "The core knows no module" or "Modules depend only on the core". Say how it is checked, for example by an architecture test in the nightly suite; a rule nobody checks is a wish. Rules of the plan, such as who usually plans where, belong to the way of working, not here.

# Product

What the product should be able to do, by themes. This part has no fixed chapters: its themes are the product's own, and the user [handbook](handbook.md) is built from it.

Fill it theme by theme. A theme is a folder with a README of type [Domain](../../packages/software/domain.md): the [concepts](../../packages/core/concept.md) it manages and the [operations](../../packages/software/operation.md) it offers, each a document of its own beside the README, with the [rules](../../packages/core/rule.md) that hold across the theme. A large theme splits into sub-themes, each a folder with its own README, linked from the theme's README. Structure it, never flat: when a folder holds many documents side by side, they belong into themes. Cut the themes by what belongs together for the user, not by the modules of the code: a theme may run across several modules, and splitting a module never moves the spec.

Example for a lending library: `lending/` with `README.md`, `loan.md` and `renew.md`; `catalog/`; `members/`. When a feature of the plan is accepted, what lasts of its target picture is written into the theme it belongs to.

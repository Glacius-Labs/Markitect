# Classes

The catalog of classes of Areas. One [class](../../../packages/factory/class.md) per document; link each from this list. A class is optional: it groups Areas and sets rules for the group, and an Area without a class is an ordinary module that depends only on the core. A class says who usually plans there, never who may; breaking its Never is a [deviation](../rules/deviations/usual-way.md). The Admins keep the catalog.

The classes of the way of working itself, which every project has:

- [Factory](factory.md): the way of working.
- [Cross-cutting](cross-cutting.md): a rule for everyone and the observer that checks it.
- [Infrastructure](infrastructure.md): pipeline, environment, delivery, dependencies.

The classes of the product belong to the project: add them here when you set up the way of working. A product's catalog usually has a class Core, which everything else depends on and which knows no module, and a class Access, for who may see and do what; the rules of the way of working name both. Further product classes describe kinds of modules, such as a variant of entering something, or a module that only shows what holds. Stories with fixed tasks run alone in every class.

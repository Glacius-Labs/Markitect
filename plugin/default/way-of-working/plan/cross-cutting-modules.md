# A module for a rule that holds for everyone

Type: [Rule](../../core/rule.md)

## Applies to
Every module that holds a rule for everyone, such as on security or architecture, and the watch that checks it.

## Rule
Such a module connects to the modules its rule concerns through stories in those modules that depend on the rule. It never changes what another module holds: for that, a story arises in that module and depends on the rule.

## Responsible
The [Admin](../team/admin.md) usually plans there.

---
kind: std/Rule
---
# Identifier limits

- **Generated identifiers stay valid identifiers.** A generated identifier is its prefix followed by a number. Within the integer limit it has at most 20 characters (a four-character prefix and at most 16 digits), so it always meets the [[values|identifier]] format and can be passed back as an argument.
- **Audit targets are not identifiers.** An audit `target` is a plain string. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format.

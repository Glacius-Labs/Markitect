---
kind: Kind
realized: false
fields:
  Justified by: {required: true, links: [std/Goal, std/Benefit, std/Constraint]}
  Eased by: {links: [std/Quality, std/Decision, std/Constraint]}
parts:
---
# Drawback

A Drawback Definition states what one [[std/Stakeholder]] loses, must newly do or risks with the system. It is a part of the Stakeholder, under `drawbacks/` in the folder of its name.

**Justified by:** the [[std/Goal]], [[std/Benefit]] or [[std/Constraint]] that makes it acceptable. **Eased by:** what softens it, in words; a link names only what lies at Root, such as a [[std/Quality]], a [[std/Decision]] or a Constraint, since Root may not link down into a child Package.

Verified when what **Justified by** names exists and the project does not make the drawback larger than it says.

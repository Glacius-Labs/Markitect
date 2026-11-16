---
kind: Kind
fields:
  Granted by: {required: true}
parts:
---
# Role

A Role Definition states one role a caller can hold, by which an [[std/Operation]] says **Who** may call it and a [[std/Page]] **For** whom it is: who counts as it, in one sentence, named for the role. **Granted by:** how one comes to hold it, such as signing in, an owner's choice or a secret link, or that everyone holds it.

It is a part of the [[std/Rule]] that says who may call what, in the folder of its name, such as `domain/rules/roles/`. A [[std/Stakeholder]] is a group of people, not a role: one Stakeholder may hold several Roles, and a Role such as `anyone` is no Stakeholder.

Verified when the system grants the role only as stated, and every Operation and Page that names it lets in those who hold it and nobody else.

---
kind: std/Rule
---
# Attachment

A [[projects/project]] holds reservations and purchases of its members, which are its owner and its collaborators.

- Only an open project takes changes. Adding a collaborator or attaching anything to a closed project is refused with `project_closed`. Closing is final, and a closed project can still be read.
- A [[bookings/reservation]] or a [[materials/purchase]] can be attached when its member is the project's owner or one of its collaborators at the request time.
- A reservation or a purchase belongs to at most one project and stays there. Attaching one that is already attached, to this or to any other project, is refused with `already_attached`.
- A reservation can be attached in any status. Only completed reservations count in the cost.

An owner can add collaborators and attach records. A collaborator cannot act on the project as a member: a member acts on a project only as its owner, which [[authorization]] states.

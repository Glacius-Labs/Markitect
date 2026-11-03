---
kind: Interface
---
# Projects interface

What other modules may use of projects: its operations, the project and cost share records and the refusal codes below. Other modules ask projects for nothing else.

- **Operations:** [[projects/create-project]], [[projects/add-collaborator]], [[projects/attach-reservation]], [[projects/attach-purchase]], [[projects/close-project]], [[projects/get-project-cost]].
- **Uses:** [[members/members-interface]], [[bookings/bookings-interface]], [[materials/materials-interface]].
- **Records:** [[projects/project]], [[projects/cost-share]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_attached` | The reservation or purchase is already attached to a project. |
| `already_collaborator` | The member is the project's owner or already one of its collaborators. |
| `not_project_member` | The member of the reservation or purchase is neither the owner nor a collaborator of the project. |
| `project_closed` | The project is closed. |

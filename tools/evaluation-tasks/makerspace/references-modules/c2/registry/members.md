---
paths:
  - realization/members/**
---
# Registry: members

| File | Realizes | Why |
|---|---|---|
| `realization/members/members.go` | [[members/members-interface]] [[members/user]] [[members/member]] [[members/registry]] | Users and members, and what other modules may ask of them |
| `realization/members/operations.go` | [[members/register-user]] [[members/register-member]] [[members/get-member]] [[members/change-tier]] [[members/set-member-standing]] [[members/membership]] | Member operations: users, members, tiers and standing |
| `realization/members/tier.go` | [[members/tier]] [[members/membership]] | Tier fee and usage discount |
| `realization/members/views.go` | [[members/user]] [[members/member]] | JSON views of the user and member records |

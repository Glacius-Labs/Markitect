# Concept record sources

Sources preserve where the [register](../register.md) entries come from. Owner statements stay verbatim and in their original language, and any context a recorder adds is labeled. Do not edit source text. Record a change of meaning as a new or superseding register entry.

| Source | Origin | Content |
|---|---|---|
| [Owner product description](2026-10-09-owner-product-description.md) | Captured by the Concepts chat on 9 October 2026; copied on 10 October 2026 | The owner's original description of the idea |
| [Idea review](2026-10-10-idea-review.md) | Claude Code review session, 10 October 2026 | Owner correction, the confirmed message and owner dispositions |
| [Concepts chat collection](concepts-chat-20261009/README.md) | Snapshot of the Concepts chat's notes, 9 October 2026 | Owner contributions `USER-20261009-01` to `-07` as the product chat handed them over, plus the Concepts chat's structured notes C01 to C34. Most handed-over blocks end with the product chat's own classification (Einordnung, Konzeptbezug, Status). `-05` and `-07` are written in the product chat's voice. |
| [Ideas chat register](ideas-register-20261009.md) | Snapshot of the Ideas chat's register, 9 October 2026 | `IDEA-20261009-001` to `-018`, `EVAL-20261009-001` and one Historian finding |

The interpretation inside the two collections belongs to the chats that wrote them; it contains no owner decisions. This applies to the Concepts notes' "Eigene Interpretation", assumptions, limits and open questions, and to the Ideas entries' problem statements and statuses. The register's [imported collections](../register.md#imported-collections) section maps their IDs to register entries.

## Snapshot provenance

### Concepts chat collection

- **Origin:** Taken on 10 October 2026 from the untracked folder `docs/design/concepts/` in the worktree of branch `codex/concepts-historian-notes`. The files were last modified on 9 October 2026, 22:38 local time.
- **Older copy:** An older, smaller copy of four of these files existed untracked in another checkout. This snapshot is the newer and complete state.
- **Changes on import:** Five relative links, pointing to four distinct targets, were changed so that they resolve from the new location:
  - `../../architecture.md`, `../../implementation-plan.md` and `../../refinement.md` gained one `../` each;
  - `../standard-operating-model.md` became `../../../design/standard-operating-model.md`.

  The changed links are in `README.md` and `canonical-model-and-delegated-development.md`. All other bytes are unchanged.
- **Duplicate owner text:** `user-description-20261009.md` contains the same verbatim owner text as the [owner product description](2026-10-09-owner-product-description.md).

SHA-256 of the original files before import:

| File | SHA-256 |
|---|---|
| `README.md` | `2fcdb50b0a565f02685b0bbf7bc331b856ab8170d4ee03c8f77c92e797a89edc` |
| `canonical-model-and-delegated-development.md` | `9a02322a6382f8dc89ce06f700722e127752f179d8164508469a7867c14ba0d6` |
| `decision-framework-ideas-20261009.md` | `caf521bff32a73a31c0c088e115b37cb4ad88f3f804b4fbb5f3265b7ca159428` |
| `discussion-boundaries-20261009.md` | `c901d501f9b334170e31d3e1cc719de7c0a13cfb6df6e824fe12a95d5c430cff` |
| `discussion-impulse-20261009-01.md` | `3bb2767d6d108fee89fa24f542611d1b22cf74533c23ab8437833b2487baeb4b` |
| `government-as-model-governance-20261009.md` | `41d581e7dbbeccfd1a0b2d6cb3180a3a62c59b44ad23ba7066ebf5ab3dad4102` |
| `government-evaluation-mandate-20261009.md` | `e41640db03905b8aced427a7100885c0f7eee0cb3c314195cb754c698b1c542b` |
| `historian-intake.md` | `28cee2b611e26427604c7000a7c5d1db34081f03105b7dff6389bc67743dfbc8` |
| `operational-overview-ideas-20261009.md` | `ae293b4d9de0dd5b176f8bb58b53ff3e9d62ed7f7e40a4874dc7adefc96e7e0e` |
| `operational-overview-source-check-20261009.md` | `22dd08dcaa1642984ef00e2cb581458b5ec7c5b88fd4d3d224cdab1bacdd98bb` |
| `personal-agent-cockpit-20261009.md` | `c168e50067663bc3de43865a51444f9d6959d8a5a26e4de93a5eeebbb50d9636` |
| `source-snapshot-20261009.md` | `e39cc7f20ac0685a777421e87dcfa3142190708a23eab0141a5ef8d23990545d` |
| `user-description-20261009.md` | `d6fe88aaa6029439bf0b0a64bc9f06ee47f7d11f7298e4a797b643bd37cf2aa9` |

### Ideas chat register

- **Origin:** Copied unchanged from `docs/research/ideas/README.md` on branch `codex/ideas-notes-20261009`, at commit `a57c64b40ca426c8e3a7211798e381c381ecc1ab` (blob `34f61018d8ff856c9f0bcc7d8a97f9c5c617a296`). That branch was never merged.
- **SHA-256:** `4f45234f77d3e58d09f465ff3f23d124c08e4def81a3001e4c88ef347a2f51b1`

## Continuing the collection

New concepts and ideas belong in the [register](../register.md), with any owner wording added here as a dated source file. Separate, untracked or branch-only notes are what this record replaces. Their later changes would not be reflected here.

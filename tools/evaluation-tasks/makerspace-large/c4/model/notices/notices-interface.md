---
kind: Interface
---
# Notices interface

What other modules may use of notices: its operations and the notice record. Other modules also post a notice for a member: they name the member, a kind other than `manual` and a reference, and the notice is created as [[notices/posting-notices]] states. A posted notice is part of the posting operation and is not audited on its own.

- **Operations:** [[notices/post-notice]], [[notices/list-notices]], [[notices/acknowledge-notice]].
- **Uses:** [[members/members-interface]].
- **Records:** [[notices/notice]], [[notices/notice-kind]].

This module adds no refusal codes.

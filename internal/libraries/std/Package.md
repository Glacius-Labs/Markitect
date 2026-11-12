---
kind: Kind
---
# Package

A Package Definition is the anchor of a Package: the Definition that carries the folder's own name, `members/members.md` for the Package `members`. It says what the Package is as a part of the project. A folder without an anchor is no Package and only sorts what the Package above it holds.

- State what the Package is responsible for, so that no two Packages are responsible for the same thing.
- State what it keeps to itself: what the rest of the project must not rely on.
- Leave out what it contains and what it depends on. `markitect list --deps` shows both.

It is verified like every Definition: the files the Package owns do what it is responsible for, and they rely on nothing of another Package that no link of its Definitions declares.

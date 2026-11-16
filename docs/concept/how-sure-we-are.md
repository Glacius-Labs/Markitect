# How sure we are

The tool can't stop a determined agent that runs as the user on the user's machine. This concept guards against forgetting and shortcuts, which is what has been seen, and against deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec compiles before it is verified | `status` gives no list and `mark` records nothing while the spec has errors | Firm |
| Every Definition is verified at the state it is in | The record's inputs, and `mark` binding a verdict to the fingerprint the reviewer read | Firm for one step of links, the registry and what binds a Definition: a changed Rule lists what it binds, by name; deeper reach and the code's imports are not followed |
| What is committed is what was verified | The session commits the work with the record; the tool commits nothing | Seen, not prevented: `status` on any checkout lists what differs from the record, and a CI check can refuse it |
| The work doesn't stop to wait for someone | Nothing: the agents work under the owner's session's permission mode. In early trials in `auto` mode, five of six runs had no denial ([trials](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/lean-trials-20261107.md#what-the-flow-showed)) | Weak by choice: the owner is there to answer. How often manual mode asks is not known |
| The work doesn't mix with the owner's own | The builder works in a worktree of its own, and its work comes back by a merge | Good: a conflict stops the merge. What the owner did not commit is not built. In a folder without Git the work shares the owner's folder |
| The spec doesn't change during the work | The hook refuses a write under `.markitect/` by any subagent but the architect, except the registry pages; only the owner's session starts the architect | Good for subagents, not proof: it reads the tool call, and a shell command can still write. Not prevented for the owner's session or for people: a changed Definition goes back on the list |
| The agent doesn't review itself | The hook lets only the reviewer run `markitect mark`, and lets the builder start only the reviewer | Good, not proof: the hook reads the command's text and the agent's name |
| The reviewer judges for itself | Its agent file says to take the list from `status` and treat the builder's message as claims to check. What that message passes on as the user's limits it keeps to, and an item they keep it from verifying it leaves unrecorded, so the item stays on the list | Weak: the builder writes that message. Reviews given a maker's claims did no worse on planted bugs ([17 of 18 blind, 18 of 18 with claims](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/k-checker-test-20261103.md)) |
| Nothing goes to a remote | The tool has no Git, and the skills push nothing. A routine the owner set up lands by a push when its source branch tracks a remote, and a cloud routine that cannot land pushes its own branch so that its work is not lost | Catches ordinary use only |
| The reviewer really read the code | Evidence checked for existence in the folder, and hashed | Not provable |
| The code is right | The reviewer and the project's tests | Not guaranteed |
| The work has a limit | No limit on rounds; the builder stops when three rounds in a row leave the list no shorter than it has been, and the owner can stop it | Prompt text: unbounded in time and calls. There is no spending limit |

**Anyone who wants more** can run `markitect status`, `markitect check` and the project's tests where the agent can't reach them: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these.

**What Markitect does not do:** prevent a deliberate bypass; commit by itself, so what is committed matches what was verified only if the driver does it right; isolate the work from the user's machine, so the work must not read untrusted input; review a clean copy, since the reviewer reads the folder as it stands, ignored leftovers included; refuse a changed file the ignore file lists; or keep the old text of a changed Definition.

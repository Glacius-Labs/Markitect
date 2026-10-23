# Run Git workspace service

`LocalGit` owns one private service per Run and a separate actual repository per Manager. It never edits the selected user checkout. Initial capture materializes the complete checked Census, including accounted Gitignored untracked files, and preserves canonical empty Package directories as real Git tree entries.

Managers snapshot fixed commits, inspect them through actual Project preparation, review complete binary child diffs and integrate exact child commits. Corrections update the child from the parent's actual integrated baseline. A conflict aborts the merge and returns concrete feedback to the responsible Manager.

Retained candidates are self-contained Git bundles plus exact manifests. Before returning a CandidateRef the adapter reopens the bundle independently, verifies commit/tree and full object connectivity, then publishes the stored directory. Fresh instances can reopen it after all Run copies and the original source are gone. Run closure removes only verified private paths.

The controlled source fixture covers complete hierarchy, integration/repair, questions, immutable review inputs and durable state. Local Git establishes source behavior; it does not establish Docker isolation or provider support. The [plan](../../../docs/implementation-plan.md) owns those separate gates.

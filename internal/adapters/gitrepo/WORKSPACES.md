# Run Git workspace service

`LocalGit` owns one private service per Run and a separate actual repository per Manager. It never edits the selected user checkout. Initial capture materializes the complete checked Census, including accounted Gitignored untracked files, and preserves canonical empty Package directories as real Git tree entries.

Managers snapshot fixed commits, inspect them through actual Project preparation and review complete child context: textual diffs, changed paths, Git modes, full blob IDs and binary change markers. Review context carries no opaque binary patch. The parent repository already holds the fixed child commit, including its complete binary content for inspection; integration consumes that exact commit. Corrections update the child from the parent's actual integrated baseline. A conflict aborts the merge and returns concrete feedback to the responsible Manager.

Retained candidates are self-contained Git bundles plus exact manifests. Before returning a CandidateRef the adapter reopens the bundle independently, verifies commit/tree and full object connectivity, then publishes the stored directory. Fresh instances can reopen it after all Run copies and the original source are gone. Run closure removes only verified private paths.

The controlled source fixture covers complete hierarchy, integration/repair, questions, immutable review inputs and durable state. Local Git establishes source behavior; it does not establish Docker isolation or provider support. The [plan](../../../docs/implementation-plan.md) owns those separate gates.

`LinuxGit` supplies the production workspace shape: one held Git service and one owned private volume per Run, with independent Manager repositories in that volume. Native Sessions consume their volume/subpath directly. Fixed observations export bytes to a private host copy for Project inspection; product work is not synchronized through a writable Windows bind. Disposable check input and home copies use separate subpaths with verified bytes, Linux modes and explicit required directory inventories. Candidate bundles and captured external inputs are independently read back before retention.

The Linux backend requires a separately observed container proof. Local fixtures, compilation and controlled command receipts do not establish its physical mode, isolation or cleanup behavior. The direct Windows bind probe failed mode preservation and is not the production workspace path.

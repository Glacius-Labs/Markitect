# Working-tree publication

`Repository` implements `apply.Publisher` alongside source capture. Use the same
adapter instance for Capture, Preflight and Publish. A successful Capture retains
its explicit selection, immutable census, external-source digest and physical
repository identities. Preflight replays that selection and checks the original
guard and live controller before issuing an opaque permit for the entire copied
request. Apply decides artifact scope and run authority; this adapter has no run
or review state. Empty changes are valid.

Publish repeats source, controller, checkout-policy and complete index checks
before writing. It creates ordinary directories and adds, replaces or removes
only requested files. Replacements remove and recreate the directory entry, so
hard links elsewhere retain their content. This has a visible interruption gap;
multi-file publication is not atomic and partial work remains for inspection.
Rooted handles, exact names and directory identities retain acquisition's
redirect protection at the write boundary.

Git performs checkout and clean conversion in an external temporary bare
repository with an explicit closed attribute/configuration set. Owner Git
objects, index, refs and configuration are never written. Hooks and external
filters are never executed. Preflight refuses active filters, working-tree
encodings, ident expansion, legacy crlf attributes, changes to any spelling of
`.gitattributes`, case or Unicode aliases, native invalid names and modes the
target cannot retain. Split indexes are refused because the complete comparison
does not acquire their shared index dependencies. Automatic conversion of existing CRLF index content is
also refused because Git's index-sensitive conversion cannot be reproduced by
the isolated converter. Explicit text/eol attributes and normal automatic
conversion of LF content are supported.

After the last write, the controller and complete original acquisition are
checked again. Every census entry outside the request, external selected source,
HEAD, repository identity, complete index bytes and selected checkout policy must
still agree. Every requested path is classified against its before/after content
in Git clean form. Any failed final fence is unknown, including when all requested
paths currently match. Errors return known per-path observations too.

`Inspect` works with a fresh adapter and the original journaled request. It reads
and classifies every requested path, never writes or replays, and always reports
an unknown publication outcome: current content cannot prove an earlier write or
controller fence. Fixture tests use temporary Git repositories only; they are
source evidence, not a real Apply or provider acceptance.

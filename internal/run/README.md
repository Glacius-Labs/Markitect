# Closed hierarchical Run

`Executor.Execute` executes one Assessment, Implement or Verify with the complete checked Project hierarchy. It opens every Manager environment before ordinary work, gives each its own Git copy, and uses direct child delegation, asynchronous questions, exact child review and upward Git integration. Completed Managers remain reachable in their ordinary conversation.

## Collaboration contract

Run keeps one explicit assignment state per Manager (`assignment.go`). Each change follows a named event: a parent assigns a task, the Manager receives it, judges it, its handoff is fixed, the parent reviews it, a correction reopens it, or a late finding changes it. A judgment binds the received task (with its mandatory Findings) and the assessed state: the Implement working-copy commit at the time of the judgment, otherwise the fixed input candidate.

The Manager tool set is fixed: `delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes` and `append_note`. Run sets the acting Manager and the origin of every Finding. A refused call returns its concrete reason and the possible next step.

- `ask` reaches only the direct parent or a direct child; the refusal names the allowed addressees. Siblings align through their common parent.
- A message to an idle Manager carries its newly delivered events and Run's diagnostics; `await` delivers events that arrive while it works and refuses at once when nothing can arrive.
- The end of a native response is not a judgment. A response without one leaves the assignment active; Run continues it, naming the open task, only when nothing it waits for (a working child, an open question) is outstanding. Questions and answers never need a judgment, also before the first one. A continued response must act through a tool or change the working state: after two continued responses in a row without progress the Run ends technically incomplete with `ErrNoProgress` and a diagnostic naming the Manager.
- Run checks a judgment when the Manager records it and before relying on it. After a response ends, a changed assessed state invalidates the judgment with the reason "your working state changed after your judgment". Root completion checks Root's earlier judgment again on the quiet hierarchy.
- A correction is a new task: no earlier judgment applies. A late adverse judgment about a fixed handoff stays in force; the parent's acceptance no longer holds, ancestors that already handed off work again, and the nearest active ancestor loses its judgment. A model blocker cannot be corrected within the Run.

## Checks

Every Implement submission is fixed, checked and delivered to the parent with its `Check`, also when the check fails. The parent can neither approve nor integrate an invalid submission; it rejects with Findings, settles the interface or reports a model blocker. Root has no parent: Root's Implement submission or Assessment/Verify approval of a state that fails its check is refused with Root's own diagnostic. The `check` tool runs the same check on the caller's assessed state for optional self-correction. Assessment and Verify handoffs carry no check; reviewers use `check` for the fixed input.

The check is structural validation and accounting, then the declared checks placed in that Manager's copy (Assessment runs only the structural part). Declared checks additionally require obligation eligibility and directory ownership confined to the Manager subtree, including nested exact-file areas. Root runs the complete suite; checks without `requiredBy` remain Root-only. A child state that only lacks integration still outstanding elsewhere is therefore not invalid; `Check.Checks` lists the checks that ran, and a check that did not run is never reported as passed. Assessment and Verify retain their fixed input; review scratch cannot replace product bytes. Implement submission establishes no independent approval or publication.

## Limits and evidence

The shared allowance bounds all messages and questions; the request bounds Manager count, cleanup and terminal persistence separately. Partial startup, blocked adapters, cancellation, observation failures and storage/read-back failures cannot produce successful technical completion. Cleanup records actual outcomes and uncertainty; a timeout never claims a process was killed.

`gitrepo.LocalGit` supplies real isolated repositories and self-contained stored bundles, including ancestry, modes and explicit empty model directories. `runstore.Store` supplies immutable terminal records and concurrent project Notes. Candidates and Notes reopen through fresh adapter instances after temporary copies close.

The real-Git vertical fixture has Root, two siblings, a grandchild and an empty Package. Controlled Sessions execute all three mandates with crossed parent/child questions and questions after fixed handoff. Implement additionally exercises rejected-child correction, an invalid child candidate reaching its parent, actual Git merge conflict recovery, late Reject repair and repair after accepted parent integration. Separate controlled tests cover each counterexample of concept §13.3 and refused questions outside the hierarchy. Failure tests inject partial startup, blocked Exchange/Close, cleanup/storage/read-back failure and insufficient preflight limits. These are source proofs, not native provider or container-isolation evidence.

The production Git workspace shape is `gitrepo.LinuxGit`: one held Git service per Run and private volume-backed Manager copies. Check input/home copies preserve explicit directory inventory as well as bytes and modes. Provided native Session observations bind the runtime profile identity, image and rendered harness digest; controlled observations remain transport evidence. Physical Linux/container and provider loading proofs retain their separate source-bound results. The [delivery plan](../../docs/implementation-plan.md) owns current scope.

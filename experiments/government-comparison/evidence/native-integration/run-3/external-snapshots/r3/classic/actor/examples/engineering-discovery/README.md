# Engineering-style discovery pilot

This is a synthetic, one-repository exercise of a human-led discovery workflow. The C# files are opaque UTF-8 evidence; neither the example nor Markitect derives their meaning. `context-run.yaml` fixes a task and explicitly selected evidence paths. A separate dossier records observations, a counterexample, a provisional candidate, and a human decision tied to exact hashes. The candidate and dossier stay outside the copied Project and its Areas.

The example directory itself is a small positive Markitect Project for the normal working-tree gate: `markitect check --repo examples/engineering-discovery`. The nested `project-template/` is copied to its own repository for the immutable ContextRun replay below.

The fixture does **not** claim that the pattern is a real team preference, that an AI successfully inferred it, or that a human approved it. The test creates a synthetic `accepted` decision solely to prove that the helper checks byte bindings and still reports that it cannot authenticate a reviewer or perform adoption.

## Replay the fixed evidence selection

Copy `project-template/` to a dedicated temporary directory, initialize it as its own Git repository, and commit the files. The run manifest must be present in the selected commit:

```powershell
$project = Join-Path $env:TEMP 'markitect-engineering-discovery-project'
$dossier = Join-Path $env:TEMP 'markitect-engineering-discovery-dossier'
Copy-Item -Recurse .\examples\engineering-discovery\project-template $project
Copy-Item -Recurse .\examples\engineering-discovery\dossier-template $dossier
Push-Location $project
git init
git add .
git -c user.name='Discovery Example' -c user.email='example@invalid' commit -m 'Freeze selected discovery evidence'
$revision = git rev-parse HEAD
Pop-Location

go run ./cmd/markitect context --repo $project --revision $revision --run context-run.yaml > .\context-report.yaml
```

The report contains the tool version and digest, full revision, snapshot digest, ContextRun manifest hash, selection hash, and a hash/status for each selected path. Copy those exact values into `evidence.yaml`; the helper uses the recorded tool identity to recompute the same context digest. Review the captured source excerpts before using them. For another repository, create a separate run and keep its source identity and hashes separate in the outer ledger. Review comments or conversations are eligible only as explicitly exported redacted UTF-8 evidence with an immutable source URL/ID and date; this example makes no provider calls.

## Create and review a candidate dossier

Keep the dossier next to, not inside, the copied Project. The `dossier-template/` files show the fields and candidate structure; replace their `FILL_IN` values from the ContextRun report. Evidence IDs must map to exact `sources` entries. Candidate text must list at least one supporting ID and one counterexample ID in the matching sections. Record the context snapshot, run manifest hash, selection hash and source byte hashes in `evidence.yaml`. Before a human decision, compute `candidateHash` and `evidenceHash` as `sha256:` plus the lowercase SHA-256 of the exact file bytes and write them into `decision.yaml`.

Run the deterministic example helper after review:

```powershell
go run ./examples/engineering-discovery/check-discovery.go `
  --repo $project --revision $revision --run context-run.yaml `
  --evidence (Join-Path $dossier 'evidence.yaml') `
  --candidate (Join-Path $dossier 'candidate.md') `
  --decision (Join-Path $dossier 'decision.yaml')
```

It checks fixed-snapshot identities, exact selected-input hashes, evidence IDs, decision status, and candidate/evidence byte digests. Evidence excerpts must be selected and redacted by a person before sharing; the helper does not scan or classify their contents. Any changed selected source, candidate, or evidence ledger requires fresh hashes and a new decision record. The helper is read-only. Even an `accepted` record only makes a candidate eligible for a **separate** adoption review; the helper cannot authenticate the named reviewer or adopt policy.

## Adoption remains an ordinary Project change

Only after an explicit human acceptance, use the [Constitution Change workflow](../../internal/authoring/resources/workflow-constitution-change.yaml) on a normal Project branch. Put authoritative resources under their configured Areas. If the accepted rule needs a new vocabulary, define/load its Domain before instances; if it is genuinely reusable across independently maintained Projects, package the Domain and resources and pin/select that package explicitly. Keep descriptive guidance, normative rules, and source-analysis checks distinct. Run the Project's normal formatting, `check`, selected `context`, `impact`, configured checks, and semantic review. Discovery never writes canonical resources or changes the graph.

The outer dossier can combine multiple repository runs without merging their provenance. The first pilot deliberately uses one repository so owners can assess review usefulness and effort before considering broader automation.

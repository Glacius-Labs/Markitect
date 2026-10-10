# GitHub repository metadata command adapter

This standalone executable is a read-only consumer of Markitect's normalized
semantic-model protocol. It compares exactly selected `Repository` resources
with exact declared files containing sanitized GitHub REST response captures.
It does not make HTTP requests, load credentials, contact GitHub, or write
repositories.

## Bounded contract

Each mapped canonical resource must have top-level `data.fullName` in
`owner/repository` form and a non-empty `data.defaultBranch`. Those values are
the only desired values. The complete command configuration declares the
exact capture input and maps the canonical identity to it; the mapping does
not repeat expected state:

```yaml
spec:
  adapters:
    - name: github-repository-metadata
      type: command
      version: markitect-github-repository-metadata/v0.1.0
      config:
        inputs:
          - evidence/github/markitect.json
        observe: [markitect-adapter-github, observe]
        plan: [markitect-adapter-github, plan]
        verify: [markitect-adapter-github, verify]
        parameters:
          repositories:
            - resource: engineering/github.example.org/v1alpha1/Repository/markitect
              evidenceFile: evidence/github/markitect.json
```

Each mapped capture must appear exactly once in `config.inputs`; each exact
resource and evidence path may appear only once in the adapter mapping. The
adapter rejects unknown parameter fields and paths outside the staged input
directory. It reads only mapped files and rejects repeated JSON keys in the
capture envelope and response body, including keys that differ only by case.
That conservative rule avoids ambiguity from Go's case-insensitive struct-field
matching rather than claiming that JSON keys themselves are case-insensitive.

Each evidence file is one bounded JSON capture envelope:

```json
{
  "apiVersion": "markitect.github.example.org/repository-capture/v1alpha1",
  "request": {
    "method": "GET",
    "path": "/repos/Glacius-Labs/Markitect",
    "apiVersion": "2026-03-10"
  },
  "response": {
    "statusCode": 200,
    "body": {
      "full_name": "Glacius-Labs/Markitect",
      "default_branch": "main"
    }
  }
}
```

This capture envelope records the request path and response status alongside
selected response-body fields. GitHub's `Get a repository` REST endpoint is
`GET /repos/{owner}/{repo}`; its response includes `full_name` and
`default_branch`. The adapter pins captures to API version `2026-03-10`, the
version currently shown in GitHub's REST reference. The capture is still
caller-supplied evidence: its envelope does not authenticate the origin, prove
that the request was sent, or prove what permissions the caller had. See the
[official endpoint reference](https://docs.github.com/en/rest/repos/repos?apiVersion=2026-03-10).

The requested path and response `full_name` must match the canonical
`data.fullName`; a mismatch makes evidence incomplete. Non-200 statuses,
including `403` and `404`, are incomplete evidence, so missing or unauthorized
objects cannot pass. A completed capture with a different `default_branch`
produces a deterministic `default-branch-drift` finding. `observe` and `plan`
may be complete while reporting drift; `verify` reports `failed` when drift
remains. Plans contain no operations and this adapter does not support `apply`.

Each observation includes the exact canonical scope, the captured repository
identity and default branch, the request path, the evidence kind/API version,
and a SHA-256 digest of each exact capture file. Planning and verification
replay the staged fixed inputs. They do not establish live or continuing
GitHub state. Capture provenance, HTTP authorization, API pagination/rate
limits, GitHub Enterprise, other repository settings, and mutation are outside
this adapter's claim.

Each capture file is capped at 1 MiB; the full adapter request is capped at
10 MiB as a prototype implementation bound. The request cap is local to this
executable, not a Markitect protocol or scale guarantee.

## Build and test

From the Markitect repository root:

```powershell
go test ./src/cmd/markitect-adapter-github -count=1
New-Item -ItemType Directory -Force .artifacts/adapters | Out-Null
go build -o .artifacts/adapters/markitect-adapter-github.exe ./src/cmd/markitect-adapter-github
```

The executable can be configured as any `type: command` adapter using the
frozen `adapter-request`, `adapter-result`, and `adapter-plan` v1alpha1
protocols. Its process test builds and runs the executable with a real
`app.AdapterRequest` and `app.SemanticModel` serialization, including
snapshot/configuration/status, Domain, relationship, resource identity,
label, and source fields. It is built separately; this change does not add a
GitHub-specific Markitect CLI command or change Core, Project syntax, or
shared adapter DTOs.

# Government G1 example

This fixture exercises the read-only Government model, bounded native observation, and conservative plan. It gives the order and inventory requirements exact identities, recursively assigns child Areas beneath a root integration Area, grants child work through prior subset mandates, and gives one crosscutting Ressort review scope. Responsibilities answer who owns a subject; technical capabilities are deliberately empty because language and test tools do not create responsibility.

`docs/requirements.md` realizes both subjects. The reservation subject also maps to its implementation and test files, while each path has one declared Area writer. `misc/unassigned.md` is an actual unmodeled file and should appear as unknown. The operational `order.yaml` and `negative-order.yaml` are also observed as unknown project files; the README is explicitly excluded from the observation boundary. Both order files must remain readable in the declared inventory because the CLI verifies each request against its captured bytes.

From the repository root, inspect the model and actual fixture files:

```powershell
go run ./cmd/markitect government --repo examples/government --config government.yaml --action inspect
```

The checked orders bind the current Constitution digest, `sha256:9c94d00054effeff3477850f1872c250d0ea7b9f5602e384fe6a6853148fe70a`. If a canonical definition changes, rerun inspect and update both orders to its new digest. Plan the bounded order:

```powershell
go run ./cmd/markitect government --repo examples/government --config government.yaml --action plan --order order.yaml
```

The concrete negative authority case uses the same selected constitution and real reservation identity but asks the inventory child to amend the accepted model:

```powershell
go run ./cmd/markitect government --repo examples/government --config government.yaml --action plan --order negative-order.yaml
```

Expected inspection: no structural findings; two modeled requirements; one shared file for both subjects; two additional files for reservation release; one writer per managed path; the declared root and README exclusion; and `misc/unassigned.md` visible as unknown alongside the two operational Order files. The output is still useful when the CLI exits 1: that exit reports the incomplete coverage caused by the deliberate unknown files. The request's inspected Constitution digest must match the order before a plan can proceed.

Expected positive plan: `planned-scoped`, with a read-only proposal covering the reservation subject and the shared file's related cancellation subject, plus inventory and root integration review. Since the shared documentation path is written by root, it is routed to root. Survey coverage remains incomplete because the unknown files stay visible. Expected negative plan: `blocked` with `authority.missing` findings because the inventory child mandate permits implementation and review, while model amendment remains with root. A plan is not execution, independent review, a vote, acceptance, or host takeover.

The example has no runner, candidate lifecycle, review isolation, voting record, or protected host-application result. Those are G2 work and must be demonstrated end to end before claiming them.

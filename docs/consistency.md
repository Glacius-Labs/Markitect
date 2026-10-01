# Opt-in factual consistency

`Project.spec.consistency.functionalPredicates` names predicates for which one subject can have only one value in the selected snapshot. A local resource may declare `spec.assertions` with `subject`, `predicate`, `value`, `source`, and `quote`. The source must also be listed in the resource's `spec.files`, so ordinary file access, context, and impact stay explicit. The quote must occur exactly once in that source. A deterministic finding identifies conflicting values, both owners, paths, and source lines. No assertions are compared unless the Project opts in and lists their predicate.

```yaml
spec:
    consistency:
        functionalPredicates: [owner]
```

```yaml
spec:
    files: [docs/operations.md]
    assertions:
        - subject: release-approval
          predicate: owner
          value: operations
          source: docs/operations.md
          quote: Operations owns release approval.
```

This detects different declared values for one functional relationship, missing source files, and quotes that disappeared or became ambiguous. The quote binds a declaration to a visible source; Markitect cannot prove that the declared value faithfully interprets the prose. Owners review assertion changes and settle any disputed meaning. `check` and `verify` treat deterministic conflicts as diagnostics. An optional AI reviewer may propose other inconsistencies, but its report remains advisory evidence and cannot change a declared fact or supply human acceptance.

The executable [contradictory fixture](../examples/consistency-conflict/README.md) declares `release-approval/owner=operations` and `release-approval/owner=review` in two quoted sources. Running `markitect check --repo examples/consistency-conflict` must fail with `consistency.conflict` and identify both owners and source lines. The fixture demonstrates a conflict in declared assertions; it does not establish which owner is correct.

Free-text contradictions, conditional rules with different scopes, temporal qualifications, synonyms, implicit ownership, and disagreements not represented by the same subject and functional predicate remain outside this MVP. Do not add a functional predicate for a naturally multivalued relationship. Existing graph, contract, binding, and output checks continue to validate their structural relationships separately.

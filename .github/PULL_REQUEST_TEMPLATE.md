## Package

- **Package ID:** <!-- for example ARCH-04; the title starts with it -->
- **Zone:** <!-- from the roadmap's zone table; name any out-of-zone edit and who approved it -->
- **Branch:** <!-- dev/…, fix/…, docs/… or exp/…, started from current origin/main -->

## Summary

<!-- Describe the user-visible behavior or product documentation change. -->

## Acceptance

<!-- Copy each acceptance line of the package from docs/work-items/backlog.yaml and give its evidence. -->

- [ ] <!-- acceptance line: evidence -->

## Validation

- [ ] `git show --stat` (or `git diff --name-only origin/main HEAD`) lists only the intended files.
- [ ] I ran the checks relevant to this change and list the results below.
- [ ] I updated canonical sources and regenerated derived schemas or views when needed.
- [ ] Examples remain executable when affected.
- [ ] Documentation reflects the current behavior and distinguishes source changes from released distributions.
- [ ] The change preserves explicit ownership and inputs, deterministic diagnostics, and fixed-snapshot evidence where applicable.

Checks run:

```text
<!-- e.g. go test ./..., go vet ./..., go run ./src/cmd/markitect schema --repo ., go run ./src/cmd/markitect check --repo examples/minimal -->
```

## Findings and owner decisions

<!-- Anything the integrator or owner should know or decide. Write "none" if there is nothing. -->

See [CONTRIBUTING.md](https://github.com/Glacius-Labs/Markitect/blob/main/CONTRIBUTING.md#pull-requests-and-parallel-work) for the pull-request rules and the full development checks.

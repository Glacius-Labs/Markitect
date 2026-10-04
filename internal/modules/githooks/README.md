# Git Hooks module

This private module checks only exact hook entrypoints selected by its own configuration and explicitly supplied inputs. Host supplies the normalized Core semantic model, the decoded module config, bytes keyed by exact repository path, and path-to-owner facts. The module performs no filesystem access, Git lookup, command execution, or content interpretation.

## Configuration

The closed configuration version is `markitect.example.org/git-hooks-check/v1alpha1`.

```yaml
apiVersion: markitect.example.org/git-hooks-check/v1alpha1
hooks:
  - name: pre-commit-tests
    stage: pre-commit
    path: .githooks/pre-commit
    digest: REPLACE_WITH_64_LOWERCASE_SHA256_CHARACTERS
    owner: development/Rule/repository-checks
  - name: custom-audit
    stage: custom
    path: scripts/hooks/audit
    digest: REPLACE_WITH_64_LOWERCASE_SHA256_CHARACTERS
    owner: development/Rule/repository-checks
```

Each configured entrypoint requires a lowercase SHA-256 over the exact supplied bytes. Replace the digest placeholder with the owner-reviewed byte hash. Owner is the exact Core resource identity key expected to manage the path; Host also supplies the explicit ownership fact for that path. Supported labels are `pre-commit`, `pre-push`, `commit-msg`, and `custom`. A custom entrypoint still names one exact path.

An empty `hooks` list returns `not-configured` with an informational finding. It never reports a pass or claims that the repository has no hooks.

## Result and limits

`Check(Input)` returns deterministic findings for missing supplied bytes, stale configured digests, absent owner resources, and missing owner links. It reports the supplied snapshot/model identities and sorts findings by path and finding identity. Inputs are bounded to 128 configured paths, a 1 MiB config, and 1 MiB per configured hook artifact.

The supplied Core model must be structurally valid and carry fixed snapshot/model digests. A pass establishes only that the configured path bytes were supplied, any configured digest matches, and the explicit owner link resolves. It does not establish that Git installs or invokes the hook, that a hook is executable, that its script is safe or correct, or that it runs successfully. The module does not inspect `.git/hooks`, Git's `core.hooksPath`, hook-manager configuration, or shell semantics.

Host/runtime integration and repository hook entrypoints are separate work. No current hook set is inferred from an empty config.

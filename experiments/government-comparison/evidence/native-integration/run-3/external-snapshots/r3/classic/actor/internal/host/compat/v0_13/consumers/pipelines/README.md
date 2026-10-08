# Pipelines module

This private module checks only explicitly configured GitHub Actions and Azure DevOps YAML paths supplied by Host. It consumes a normalized Core model, its own closed config, exact artifact bytes, path-owner facts, and optional named check facts. It does not discover pipeline files or invoke a CI provider.

## Configuration

The closed configuration version is `markitect.example.org/pipelines-check/v1alpha1`.

```yaml
apiVersion: markitect.example.org/pipelines-check/v1alpha1
pipelines:
  - name: github-tests
    provider: github-actions
    path: .github/workflows/tests.yml
    digest: REPLACE_WITH_64_LOWERCASE_SHA256_CHARACTERS
    owner: development/Rule/repository-checks
    expectedChecks:
      - name: go-tests
        yamlPath: /jobs/build/steps/0/run
  - name: azure-tests
    provider: azure-devops
    path: azure-pipelines.yml
    digest: REPLACE_WITH_64_LOWERCASE_SHA256_CHARACTERS
    owner: development/Rule/repository-checks
    expectedChecks:
      - name: dotnet-tests
        yamlPath: /steps/0/script
```

GitHub Actions paths must be exact YAML files under `.github/workflows/`; Azure DevOps paths are explicit exact YAML files. The module never searches either location. Each configured pipeline requires a lowercase SHA-256 over the exact artifact bytes. Replace the digest placeholder with the owner-reviewed byte hash. The expected owner is a Core resource identity key and must also appear in Host-supplied ownership facts for that exact path.

Each `expectedChecks` entry names a Host-supplied `CheckFact{Name, Reference}` and one exact RFC 6901-style YAML pointer. The pointed-to node must be a unique string scalar, and its decoded scalar value must equal the supplied reference as an exact Go string. No command parsing or normalization occurs. Unsupported aliases, missing or ambiguous mapping keys, non-string values, and invalid pointers are reported.

An empty `pipelines` list returns `not-configured`, never a successful verification claim. If `expectedChecks` is empty, only configured path presence, required digest, and owner linkage are checked.

## Result and limits

`Check(Input)` returns deterministic findings for missing supplied bytes, stale digests, owner-resource/link failures, absent Host check facts, invalid YAML and unavailable/mismatched literal references. It reports the supplied snapshot/model identities and `configDigest`. That digest is SHA-256 over deterministic JSON for the closed typed config, with pipelines sorted by path and name and each expected-check list sorted by name and YAML pointer, so YAML formatting or list order does not change the binding. Inputs are bounded to 128 pipeline files, 128 expected references, a 1 MiB config, and 2 MiB per configured pipeline artifact. Configured paths that collide under portable case folding are rejected.

A matching scalar proves only that the exact configured YAML location contains the Host-supplied literal. It does not prove the step executes, is reachable, runs on a relevant event, succeeds, or has complete coverage. The module does not evaluate GitHub Actions expressions, Azure templates/tasks, conditions, matrices, external actions, shell commands, or provider state. A configured reference is a narrow textual assertion, not a universal CI DSL.

Host/runtime integration and live CI execution evidence are separate work.

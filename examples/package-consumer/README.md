# Content package consumer

This fixture vendors the exact `review-guidance` ZIP built by `markitect pack` from fixed source commit `f6d711763d93c6f8bed2b43d5ff5cf6b16458bf5` and pins its SHA-256 in [markitect.yaml](markitect.yaml). A consumer-owned Skill wraps the exported package Workflow. The package also contains unexported Skill and Text resources and a declared text input.

From the Markitect source checkout, inspect the package export directly:

```sh
go run ./src/cmd/markitect-legacy context --repo examples/package-consumer --package review-guidance --namespace review --kind Workflow --name review-change
```

Inspect the consumer-owned wrapper and its imported dependency:

```sh
go run ./src/cmd/markitect-legacy context --repo examples/package-consumer --namespace consumer --kind Skill --name local-review-entry
```

The archive was built by the CLI from a fixed Git commit of the neutral package example. To update it, repeat the pack steps in [the package source README](../content-package/README.md), review the source and resulting bytes, copy the ZIP to `.markitect/packages/`, then update the consumer pin’s version, source, archive path, and SHA-256 together. A changed archive fails closed against the current pin.

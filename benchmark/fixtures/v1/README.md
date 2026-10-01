# Rollback review example

This synthetic fixture shows a small rollback review graph: a Skill needs a Contract, a Project binds that Contract to an Agent, and the Agent and Workflow refer to typed content. The Agent also declares an ordinary UTF-8 input file. See [the docs router](docs/README.md).

From the standalone Markitect repository root, run the built binary:

```powershell
.\bin\markitect.exe check --repo examples/minimal
.\bin\markitect.exe context --repo examples/minimal --kind Skill --name rollback-review --namespace sample
.\bin\markitect.exe render --repo examples/minimal
```

These commands read the working tree; `render` checks the checked-in views. Immutable `--revision` snapshots are read from the whole Git repository tree, where Markitect expects `markitect.yaml` at the repository root. To check this fixture at a fixed revision, copy `minimal` into its own Git repository, commit it there, and run the binary with `--repo .` from that repository root.

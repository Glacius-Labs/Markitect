# Check adapter

`runtimeconfig.Prepare` supplies an acquired `container` check profile to `MetadataPreparer`. The adapter validates its closed versioned fields and observes the exact pinned image and whole executable file without starting them. `ContainerProfileFromPrepared` projects only complete matching evidence. Host `process` profiles are retired.

```yaml
checkProfiles:
  tests:
    container:
      profileVersion: 1
      engineExecutable: /usr/bin/docker
      imageID: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      executable: /usr/local/go/bin/go
      argumentPrefix: []
      env: ["GOTOOLCHAIN=local", "GOCACHE=/home/agent/.cache/go-build"]
      limits: {milliCPUs: 1000, memoryBytes: 268435456, processes: 64}
      maxOutputBytes: 1048576
```

`CandidateRunner` selects prepared profiles for declared Project checks, inspects the exact fixed candidate and delegates literal requests to `Runner`. The Runner creates an independently verified disposable input copy and empty home, runs the exact image with finite resources and combined output, and closes both copies after physical command termination. It records image/tree, start/exit/code, timeout, output truncation, uncertainty and copy closure. Known nonzero exit is a failed check; startup, observation and cleanup errors are technical failures.

`DisposableCopies` is local to this consumer. `LocalCopies` uses private host directories and read-back of actual bytes and modes. The native bind transport must separately prove that the engine sees those same directories. Windows executable-bit preservation is not established; requests that need that mode require a POSIX materializer. Source peers establish receipt handling, not container behavior.

The existing separately authorized smoke opt-in is `MARKITECT_CHECK_WORKSPACE_SMOKE=run-check-workspace-smoke`, with explicit engine executable and preexisting pinned `MARKITECT_CHECK_WORKSPACE_CHECK_IMAGE`. That image must provide `/usr/local/bin/node`. The finite two-command smoke verifies actual bind input/home isolation, byte preservation, known pass/nonzero exit, bounded output, directory deletion and absence of labeled command containers. It starts containers and must not run as part of ordinary source tests. Its execution does not establish native provider or Apply qualification.

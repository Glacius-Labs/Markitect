# Check adapter

`runtimeconfig.Prepare` supplies an acquired `container` check profile to `MetadataPreparer`. The adapter validates its closed versioned fields and observes the exact pinned image and whole executable file without starting them. `ContainerProfileFromPrepared` projects only complete matching evidence. Only `container` check profiles exist.

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

`DisposableCopies` is local to this consumer. `copyMaterializer` uses the actual Run Copy's `Materialize` operation to create a non-Git scratch tree and an empty home in the existing Run Git service. Linux volume/subpath locations are passed directly to the pinned check container; the project image needs neither Node nor Git for materialization. Partial handles close even when creation fails. The Run service also owns final failure cleanup.

Windows Docker Desktop mapped host files with modes 0644 and 0755 to 0777 in the bounded native probe, so direct host bind copies are not the production path. The Linux volume consumer requires separate actual proof of modes, empty directories, bytes and closure. Source command peers verify receipt/error handling only. Native execution and provider/Apply qualification remain separate evidence under the [plan](../../../docs/implementation-plan.md).

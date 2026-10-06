# .NET projector

This Core-only implementation package evaluates candidate .cs and .csproj
bytes against the exact paths and project-owned Kind guidance supplied by the
Host. It does not generate C#, map Kind names to .NET conventions, interpret
canonical YAML, or access the filesystem.

A returned result has status candidate-unverified: its paths and bytes passed
this package's bounded shape checks only. The Host must run the declared checks
and record their evidence. Missing Kind guidance is an explicit escalation, and
an empty candidate is incomplete rather than a converged empty result.

# Makerspace reference implementations in the module layout

Hidden reference implementations of the Makerspace program per step, laid out as the step's own model says: one Go package per module under the module root, named after the module (`policies`, `members`, `equipment`, `billing`, `training`, `bookings`, `materials`, `safety`, `reporting`, `protocol`), with `cmd/makerspace` as the entry point and a `README.md`. They are the fixed starts for the small-change tests, where a change must stay small. Assessor material, like the [old references](../makerspace-reference/README.md); no writer receives it.

## Relation to the old trees

[`tools/makerspace-reference/`](../makerspace-reference/README.md) holds the same program as one package, `internal/ms`, written before the module cut was decided. It stays as it is, for the separate case where a change meets a program that is not laid out as its model says. Each tree here has the same behavior as the old tree of the same step: split by moving the code, not by rewriting its logic. What differs is the layout and what the split forces:

- Each module owns its records and offers the others a small Go API. A member's credit balance lives in `billing`, which owns credit; `bookings` offers `ActiveReservationOverlaps`, `CompletedSessions` and `LockoutsOf`; `equipment` declares what it needs from bookings (`equipment.Needs`) and `protocol` supplies it.
- A module describes its operations (name, permissions, arguments, rules) with the vocabulary of `policies`; `protocol` runs the checks every operation shares, keeps the audit log and wires the modules. Area modules check no roles, ownership or audit themselves.
- `lockMachine` belongs to `bookings` (the model puts lockouts there), not to `safety`.
- The tests that send request lines are black-box tests of the whole program and live in `protocol`; the pure helpers are tested in `policies` and `reporting`.
- The old `peekID` (unused) is gone.

## Relation between the steps

Each tree is the previous one plus the delta of the old trees, applied to the module that owns it. `c7/` is identical to `c6/` (no behavior change) and `c9/` to `c8/`, as in the old set.

| Tree | Requirements | Note |
| --- | --- | --- |
| `w0/` | `w0.md` | 37 operations, four policies |
| `c1/` … `c6/` | `c1.md` … `c6.md` | each tree is the previous one plus that delta |
| `c7/` | `c7.md` | identical to `c6/` |
| `c8/` | `c8.md` | |
| `c9/` | `c9.md` | identical to `c8/` |

The owner's registry pages for `w0/` to `c8/` are in [`tools/evaluation-tasks/makerspace/references-modules/<step>/`](../evaluation-tasks/makerspace/references-modules): `registry.md` for Root (go.mod, README and the `policies` package, which belong to the project itself) and `registry/<package>.md` for the nine Packages. A Package's `paths:` name its module folder (`protocol` also `cmd/makerspace`); every file has one row with the Definitions of that step's model it realizes. The tree is placed at `realization/` in the repository.

## Build and test one step

```text
cd c8
go build ./cmd/makerspace
go vet ./...
go test ./...
```

Each step is its own Go module (`module makerspace`, standard library only), so the root module's `go build ./...` never includes it. `go list -deps` of each package shows only the modules the step's architecture lists for it.

## What still holds from the old README

The trees were written only from the neutral requirements and are checked by the hidden [Makerspace oracle](../makerspace-oracle/README.md) with the same cases as the old ones; the readings chosen where the text allows more than one are the old ones, in [ambiguities.md](../makerspace-reference/ambiguities.md). The oracle-family mutants of the old set apply to the old trees only.

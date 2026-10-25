# Makerspace reference implementations

Hidden Phase C reference implementations of the Makerspace program per step, written only from the [neutral requirements](../comparison/phase-c/requirements/README.md), to calibrate the oracle. Assessor material; no writer receives it.

Each step is its own Go module (`module makerspace`, standard library only) with one full tree, so the root module's `go build ./...` and `go vet ./...` never include it.

| Tree | Requirements | Note |
| --- | --- | --- |
| `w0/` | `w0.md` | 37 operations, four policies |
| `c1/` … `c6/` | `c1.md` … `c6.md` | each tree is the previous one plus that delta |
| `c7/` | `c7.md` | identical to `c6/` (no behavior change) |
| `c8/` | `c8.md` | |
| `c9/` | `c9.md` | identical to `c8/` |

Build and test one step: `cd c8 && go build ./cmd/makerspace && go test ./...`.

`ambiguities.md` lists readings chosen where the text allows more than one.

Mutants: `mutants/<family>/` holds `mutant.patch`, `STEP` (the tree it applies to) and `NOTE.md`; `mutants/apply.sh FAMILY DEST` copies the step tree and applies the patch. One mutant per oracle probe family (30). P-RULE-ORDER is not caught by the unit tests of its own tree.

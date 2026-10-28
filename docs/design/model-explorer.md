# Model explorer

Status: design proposal of the [Cockpit lane](../workstreams/cockpit.md), 27 October 2026. Nothing is built. The owner asked for a read-only model explorer as the Cockpit's first slice and for this design before any code.

## Goal

A person opens one window and understands a canonical model: what is in it, how its parts depend on each other, who refers to what, which files realize it, what each Manager will see, and what a change touches. The explorer only reads. It writes no file, starts no Run, spends no provider usage and needs no container engine. It runs natively on Windows, Linux and macOS, like `markitect check`.

## How it is used

```text
markitect explore [--repo .] [--model PATH] [--dependency PATH ...] [--revision REV] [--listen 127.0.0.1:0] [--no-open]
```

- The model options are those of `markitect check`. Without `--revision` the explorer reads the working tree, so a model can be explored while it is being written.
- The command prepares the model, serves the page on a loopback address and opens it in an app window of the installed Edge or Chrome (no tabs, no address bar). The address is printed; `--no-open` only skips the window. If no browser is found, the printed address is opened by hand.
- *Reload* in the page prepares the model again. There is no file watching.
- Closing the command ends the explorer.

## What the person sees

One hub and four areas, in the card style and navigation the owner asked for on 21 October: the hub leads into an area, Escape leads one level up, the digits 1 to 4 enter the areas, and each place has a title, one sentence and nothing more until the person asks for it.

### Hub

- The project: model ID, canonical model digest, source (working tree or revision), check status.
- Counts: Packages, Definitions, Kinds, realized files.
- What needs attention: the strongest lint warnings and Package cycles, at most five, each a link.
- One tile per area with a one-line summary.

When preparation fails, the hub shows the diagnostics, root causes first, and closes only what the failed stage leaves missing. A failed model check supplies no Definitions, so every area closes. A failed file accounting or registry still yields Definitions and Manager contexts, which is common while a model is being written: Read, Map and Organization stay open, and only *Realized by* and the file side of Changes close.

### 1. Map

The Package dependency graph.

- A node is a Package; its size is its number of Definitions; Library Packages are marked as imported.
- An arrow means that Definitions of one Package refer to Definitions of another, by a typed reference or a body link.
- Packages are placed in layers so that most arrows point down. Packages in a cycle are grouped and drawn red; edges that break the model's own architecture rules are drawn red.
- Choosing a Package shows its card: its Definitions, the Packages it depends on and that depend on it, its coupling and instability, and its warnings.
- A toggle shows the plain Package tree instead of the graph.

### 2. Read

The Definition reader, in three columns that stack on narrow windows.

- Left: Packages and Kinds as filters. Middle: the Definitions, with a search (`/`) over ID, title and body. Right: the chosen Definition.
- The Definition card shows its title, Kind and Package; its body rendered as Markdown, with every `[[link]]` opening its target; its typed fields, with every reference opening its target; *Referenced by*, every incoming reference with where it sits; *Realized by*, the files the registry links to it; *Seen by*, the Managers whose context includes it and why; and its lint warnings.
- A Kind's card also lists its fields and its instances in the model.
- Back and forward follow the reading path, like a browser.

### 3. Organization

The Manager tree drawn as an organization chart: Root on top, one card per Package Manager below its parent, each with its Package, its number of own Definitions and what `execution.yaml` declares for it: instructions, ownership areas and required checks. Agent profiles are not shown; they live in the operator's runtime profile and run configuration, outside the repository.

Choosing a Manager shows exactly what that Manager receives as static context, in the four sections of the [Manager context](manager-context.md): own Definitions, See also with the selection reasons, type contracts and child areas. It is the same selection `markitect context` reports, grouped by the same rule the harness uses.

### 4. Changes

What a change touches, between two model states: two revisions, or a revision and the working tree (by default `HEAD` and the working tree).

- Changed, added and removed Definitions, with a side-by-side difference of body and fields.
- The Definitions that refer to a changed one, directly or, on request, transitively.
- The files that realize any of them, and the Managers whose Packages hold them.

A model whose dependency manifest is given by an absolute path cannot be read at a revision, because that dependency has no revision binding. The explorer then shows the refusal plainly instead of a comparison.

## Architecture

The explorer fits the existing [layers](../architecture.md#layers) without exceptions.

| Package | Layer | Role |
|---|---|---|
| `internal/modelview` (new) | service | Pure functions over a prepared Project: the Package graph with coupling, cycles and layers; incoming references; the lint signals; the Manager tree with its contexts; the impact between two preparations. No I/O. |
| `internal/organization` | domain | Gains the rule that puts each context Definition in its section, moved from `internal/harness`, so the harness and `modelview` share one rule (a service cannot import another service). |
| `internal/adapters/explorer` (new) | adapter | The loopback HTTP server, the embedded page (`embed`), the JSON routes and the app window. |
| `internal/adapters/cli` | adapter | The `explore` verb. It reaches the explorer through a port, like the existing Apply and documentation functions, because no adapter imports another. |
| `internal/bootstrap` | composition | Builds the service and the adapter and connects them to the command. |
| `internal/tooling/release` | maintenance | The source packager takes only Go files and a few named files today, so an embedded page would be missing from the source bundle and the build would fail. It gains an allowlist for the explorer's page directory, with tests. |

- **Data.** Each preparation is the same Project preparation `markitect check` runs, for the working tree or a revision; Changes prepares twice. `modelview` reads the checked model (`model.Result`, with the Libraries' Definitions), the registry records, the file accounting, the execution declarations and the context selections from it. It does not parse the JSON report.
- **Analysis rules.** `modelview` computes edges, the Package graph, references, impact and lint in Go. It matches the Python [model analysis tool](../../tools/model-analysis/README.md) where both see the same data. They differ on purpose where Go sees more: Go reads the whole checked model, Python only the joined contexts of the report, so counts of imported Definitions can differ. Three Python signals are left out: `unused-import` and `import-cycle` need each file's own imports, which the checked model does not carry, and `direction` needs a rules file the explorer does not have. They return only if the compiler lane exposes imports per file. Whether the Python tool is then retired is decided at E2.
- **Bodies.** The server renders each body with the CommonMark library the project already uses (goldmark), set up exactly like the compiler's body parser: CommonMark only, no GFM extensions such as tables, so the structure matches the compiler's. A small inline parser consumes the compiler's body-link spans and turns each into a link, so a link opens exactly the target the compiler resolved; what the compiler treats as protected text stays plain. Raw HTML is dropped and unsafe URLs are filtered. Bodies the compiler maps specially, such as those containing NUL, are handled the same way or refused. A test pairs every span with its rendered link.
- **Routes.** All are `GET`:
  - `/`, the page and its files;
  - `/api/model`, the whole view as one JSON document, tagged with the model digest and the source's content digest;
  - `/api/model?reload=1`, prepare again;
  - `/api/changes?from=REV&to=REV`, where `to` may be `working`.

  The document holds one table of Definitions. Contexts and *Seen by* name Definitions by identity, reason and section only, so no body is repeated per Manager. A model of a few hundred Definitions fits one document; the page filters and searches locally.
- **Layout.** Graph layers and the organization chart's positions are computed in Go and tested there. The page only draws them as SVG. It is plain JavaScript modules and CSS with no build chain and no external resources.
- **Safety.** The server listens on loopback only, refuses requests addressed to another host (against DNS rebinding) and sends a strict content security policy. It has no route that writes.
- **Window.** The launcher of the retired cockpit (`window.go` on the [archive branch](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/internal/adapters/cockpit/window.go)) is ported: Edge, then Chrome, then the default browser, with the WSL case. Its host check (`host.go`) and organization chart layout are ported too.

## Tests

- `modelview`: unit tests over hand-built model values, one per rule (edges, cycles, layers, references, each lint signal, impact with and without transitivity, context sections). Where the Python tool has a test case over the same data, the Go result must match it.
- The body renderer: one test per body-link span and its rendered link, and one per refused or protected case.
- The source packager: a test that the bundle carries the explorer's page.
- `explorer`: HTTP tests with `httptest` for every route, the host check and the absence of writes; a test that the page loads no external resource.
- One end-to-end test of `markitect explore --no-open` against a fixture model in `testdata/`.
- Each slice's PR carries screenshots at 1440 and 390 pixels wide.

## Delivery

Each slice is one PR with its tests and is useful on its own.

1. **E1, Hub and Read.** `modelview` with edges and references, the command, the server, the window, the packager allowlist, the body renderer, the hub and the Definition reader. It is the largest slice.
2. **E2, Organization.** The Manager tree and each Manager's context, with the section rule moved into `organization`. It is small, because the selections already exist, and it is what no other tool shows.
3. **E3, Map.** The Package graph with coupling, cycles and layers.
4. **E4, Quality signals.** The lint signals on the map and in the reader, and the decision on the Python tool.
5. **E5, Changes.** The impact between two model states.

## Not in this slice

Editing the model, model sessions and proposals; Runs, live or closed; settings; a chat or personal agent; starting anything; an installer. Run views are the Cockpit's next slice and need run records a UI can read (see the [lane page](../workstreams/cockpit.md#what-is-reusable)).

## Open questions

1. **Areas.** Are Map, Read, Organization and Changes the right four? The recommendation is yes, with lint shown inside Map and Read rather than as its own area.
2. **Failed model checks.** A model that fails the compiler's check shows only diagnostics, because the compiler supplies no Definitions then. Should the compiler lane offer a partial model for a failed check? The recommendation is to start without it.
3. **The Python tool.** At E4, should the analysis rules live only in Go, with the Python tool retired? The recommendation is yes, decided with the compiler lane (lane 8), which proposes the same.
4. **Name.** `markitect explore`, or `markitect cockpit` from the start? The recommendation is `explore` until run views join it.

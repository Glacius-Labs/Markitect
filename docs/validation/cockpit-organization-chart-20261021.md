# Organization chart component, 21 October 2026

Package 2 of the [handoff](../development/codex-cockpit-org-chart-handoff-20261021.md),
on Main `d94a87692ecfdf44bcd491529977b97d4fa9feef`. The component consumes explicit
presentation nodes; model hierarchy and Run projection belong to their producers.

The pure `org-layout.js` lays out a top-down tree with orthogonal connectors,
reserves lateral space for the independent reviewer seat and refuses malformed
topology. Large trees initially show Root and the first department level; deeper
departments expand explicitly. Layout, collapse and visible ancestors have
dependency-free local coverage:

```text
node --test internal/adapters/cockpit/org-layout.test.mjs
```

Node is not a CI dependency. The component uses existing cockpit tokens, people
cards, lamps and CSSOM positions under the existing CSP, with pan, zoom, fit,
roving tree focus, keyboard expansion and selection, and a scrollable indented
tree below 620 px. Explicit collapse choices survive value updates. Reviewer
seats are notes with dotted attachments, not subordinate Manager tree items.

Source/layout checks are distinct from browser acceptance. Browser work is
deferred during A's frozen-source provider retest resource reservation; no
browser demonstration, screenshot or owner acceptance is claimed here. The
next package supplies the actual Organization tab and the browser integration.
No provider, container or Run was started by this work.

Independent source review found and corrected the minimum Fit scale (which
could clip very large trees), missing `aria-selected`, and reviewer notes placed
directly under the ARIA tree. Notes now describe their Manager tree item and
remain outside Manager navigation; selection is explicit to assistive tools.
All four local Node tests, module syntax, changed-page links and whitespace
checks passed. Browser and Go-based integration checks remain pending the
resource gate; this package is not yet integrated.

## Post-reservation packaging correction

The first complete cockpit Go check rejected embedded `.mjs` assets because the
source packager carries `.js` modules. The pure layout module now uses `.js`,
with its imports and exact registration updated; no packaging rule is relaxed.
The full bootstrap and architecture package tests passed during that check.
The failed cockpit packaging check is retained here and will be rerun.

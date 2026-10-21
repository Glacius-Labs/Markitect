# Organization tab preparation, 21 October 2026

Package 3 of the [handoff](../development/codex-cockpit-org-chart-handoff-20261021.md).
Prepared as a separate commit after the still-unintegrated package-2 candidate
`203bc8b8191d7f64b6f48ab4b36fb604bdc89833`, whose integrated base is
`d94a87692ecfdf44bcd491529977b97d4fa9feef`. This preparation is explicitly
authorized while A's resource reservation defers browser and heavy checks;
it is not Main integration or browser acceptance.

The Model tabs become Browse, Organization and Session. `#/organization` reads
declarations immediately from the observed project and uses the reusable chart.
Only **Resolve profiles** requests Runtime capture; its progress and refusal
stay visible, and its committed-source snapshot is labelled separately from
the working-tree declaration. A selection shows its own Definitions linked to
Browse, actual ownership areas, selection-source profiles and the project-wide
team. Missing declarations remain missing, rather than inventing defaults.

`#/settings?pkg=Package` focuses an existing override or preselects that Package
in the add selector, without creating an override on navigation. The team link
uses `#/settings?section=team`; that section explicitly applies to every Manager.
No chart interaction submits a proposal or starts provider work.

Independent source review found that Root is excluded from Settings' Package
selector. Root's profile action therefore targets the default Profiles section
with focused navigation (`section=profiles`), rather than inventing a Root
Package override. A local navigation assertion covers that exception.

Local dependency-free tests:

```text
node --test internal/adapters/cockpit/org-layout.test.mjs internal/adapters/cockpit/organization-facts.test.mjs
```

The fixtures are synthetic one-Package and large observations. They verify
explicit parent reuse, leadership, reviewer descriptions, declared/resolved
profile labels, team limits and exact navigation encoding. They are not a real
project observation or a browser demonstration of Markitect's own model.

Module syntax, local Node tests, changed-page documentation links and diff
checks are the only executed checks at preparation time. Structural/accounting,
browser keyboard/resize/390–860–1440 px screenshots, and actual one-Package and
Markitect observations remain pending the resource gate. Node remains local
only; CI gains no Node requirement. No Docker, provider or model session ran.

## Delivery after host release

Composed on actual Main `06ac30b6` (#586), retaining only this package's four
new registrations and the integrated component's `.js` path. The presentation
module also uses `.js` to match the existing source packager. The final candidate
passes the cockpit Go package, seven local Node tests, architecture CLI, working
and fixed-head structural accounting, product context, docs and diff checks.
Independent complete-diff review is required on that exact candidate before merge.

The actual embedded application displayed a checked one-local-Package fixture
(`one-package-ui-fixture`, source `34628cb40e7f62cda05b42899456b6ed33cccd88`,
with Core supplied by its pinned Markitect dependency) and Markitect's 22 local
Packages. Browse responsibility links, Package profile preselection and
project-wide team focus worked without creating a proposal. Keyboard collapse,
pan/zoom/Fit, 390/860/1440 px and an 860x400 window were exercised. The observed
390 px tab/status overlap was corrected with separate header rows and rechecked.

Resolved progress, provider-model labels, return to declarations and refusal
were demonstrated with an explicitly synthetic HTTP metadata stub; no Runtime
was contacted. These are UI checks, not provider or owner acceptance. The
finishing record will retain the combined browser screenshots. Earlier pending
statements above describe the resource-reserved preparation period.

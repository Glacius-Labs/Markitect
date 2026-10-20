// The list of recorded runs in the observed state directories.

import { api } from "./api.js";
import { h, clear, when, duration } from "./dom.js";

const outcomeClass = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" };

export async function renderRuns(app) {
  document.title = "Runs – Markitect cockpit";
  clear(app);
  const main = h("main", { id: "main", class: "runs" },
    h("h1", {}, "Recorded runs"),
    h("p", { class: "lead" }, "Runs found in the observed state directories, newest first. Opening a run replays its record; the run itself is never affected."));
  app.append(main);
  let runs;
  try {
    runs = await api.runs();
  } catch (err) {
    main.append(h("p", { role: "alert" }, `The run list could not be read: ${err.message}`));
    return;
  }
  if (runs.length === 0) {
    main.append(h("p", {}, "No run evidence was found. Start the cockpit with the state directory of an Apply (--state DIR or --run-config PATH)."));
    return;
  }
  const rows = runs.map((r) => {
    if (r.error && !r.id) {
      return h("tr", {}, h("td", { colspan: 5 }, `${r.state}: ${r.error}`));
    }
    const result = r.runOutcome || r.outcome;
    const outcome = result
      ? h("span", { class: `outcome ${outcomeClass[result] || ""}` }, h("span", { class: "lamp" }), r.runOutcome ? `${r.runOutcome}` : `${r.outcome} (no report)`)
      : h("span", { class: "outcome st-none" }, h("span", { class: "lamp" }), "no outcome recorded");
    const span = r.started && r.last ? duration(new Date(r.last) - new Date(r.started)) : "";
    return h("tr", {},
      h("td", {}, h("a", { href: `#/run/${encodeURIComponent(r.id)}` }, r.id), h("div", { class: "state" }, r.state)),
      h("td", {}, when(r.started), span && h("div", { class: "faint" }, span)),
      h("td", {}, outcome, r.publication && h("div", { class: "faint" }, `publication ${r.publication}`)),
      h("td", {}, `${r.managers} Manager${r.managers === 1 ? "" : "s"}`, h("div", { class: "faint" }, `${r.passes} Pass${r.passes === 1 ? "" : "es"}`)),
      h("td", {}, `${r.records} records`, r.error && h("div", { class: "st-fail" }, r.error)));
  });
  main.append(h("table", { class: "runtable" },
    h("thead", {}, h("tr", {}, h("th", {}, "Run"), h("th", {}, "Started"), h("th", {}, "Outcome"), h("th", {}, "Organization"), h("th", {}, "Record"))),
    h("tbody", {}, rows)));
}

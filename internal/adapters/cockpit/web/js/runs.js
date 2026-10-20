// The run history: every recorded run of the observed state directories with
// its outcome, publication, organization, source and reported usage. A run
// opens in the inspector. The cockpit neither starts nor stops a run.

import { h, clear, when, duration } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";

const outcomeClass = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" };

export function renderRuns(content) {
  const page = h("div", { class: "page" });
  const table = h("section", { class: "panel" });
  const states = h("p", { class: "note" });
  page.append(
    h("div", { class: "pagehead" }, h("div", {},
      h("h2", {}, "Runs"),
      h("p", { class: "lead" }, "Every Apply recorded in the observed state directories. Start an Apply with markitect apply; it appears here while it runs. No stop command exists: interrupting the markitect apply process ends the run incomplete with its evidence, and pausing a replay or a live view never stops a run."))),
    table, states);
  content.append(page);
  const draw = (s) => {
    if (!page.isConnected) return unsubscribe();
    clear(table);
    if (s.errors.runs) {
      table.append(h("p", { class: "empty", role: "alert" }, `The runs are not readable: ${s.errors.runs.message}`));
      return;
    }
    const runs = s.data.runs || [];
    table.append(runs.length ? runRows(runs) : h("p", { class: "empty" }, h("strong", {}, "No run evidence"), "Start the cockpit with the state directory of an Apply (--state DIR or --run-config PATH)."));
    const dirs = s.data.project?.states || [];
    clear(states).append("State directories observed: ", ...dirs.map((d, i) => [i ? ", " : "", h("span", { class: "digest", title: d }, shortPath(d))]).flat());
  };
  const unsubscribe = store.subscribe(draw);
  if (Object.keys(store.data).length) draw(store);
  else refreshState();
}

// runRows is the run table, shared with the overview.
export function runRows(runs) {
  const rows = runs.map((r) => {
    if (r.error && !r.id) return h("tr", {}, h("td", { colspan: 5 }, `${r.state}: ${r.error}`));
    const result = r.runOutcome || r.outcome;
    const run = r.health?.run;
    const outcome = result
      ? h("span", { class: `outcome ${outcomeClass[result] || ""}` }, h("span", { class: "lamp" }), r.runOutcome ? r.runOutcome : `${r.outcome}, no report`)
      : h("span", { class: `outcome ${run === "ended-without-report" ? "st-fail" : run === "running" ? "st-working" : "st-none"}` }, h("span", { class: "lamp" }),
        run === "ended-without-report" ? "ended without outcome or report" : run === "running" ? "running" : "no outcome recorded");
    const span = r.started && r.last ? duration(new Date(r.last) - new Date(r.started)) : "";
    return h("tr", {},
      h("td", {}, h("a", { href: `#/run/${encodeURIComponent(r.id)}` }, r.id), h("div", { class: "sub", title: r.state }, shortPath(r.state))),
      h("td", {}, when(r.started), span && h("div", { class: "sub" }, span)),
      h("td", {}, outcome, r.publication && h("div", { class: "sub" }, `publication ${r.publication}`), r.reason && h("div", { class: "sub" }, r.reason.length > 140 ? `${r.reason.slice(0, 139)}…` : r.reason)),
      h("td", {}, `${r.managers} Manager${r.managers === 1 ? "" : "s"}`, h("div", { class: "sub" }, `${r.passes} Pass${r.passes === 1 ? "" : "es"}`), r.head && h("div", { class: "sub digest" }, `source ${r.head.slice(0, 12)}`)),
      h("td", {}, `${r.records} records`, usage(r.usage), r.error && h("div", { class: "st-fail" }, r.error)));
  });
  return h("table", { class: "table" },
    h("thead", {}, h("tr", {}, h("th", {}, "Run"), h("th", {}, "Started"), h("th", {}, "Outcome"), h("th", {}, "Organization and source"), h("th", {}, "Record"))),
    h("tbody", {}, rows));
}

// shortPath names a directory by its last two segments; its title keeps the
// whole path.
export function shortPath(path) {
  return (path || "").split(/[\\/]/).filter(Boolean).slice(-2).join("/");
}

// usage states what the report says about provider use; without a report it
// is unknown, not zero.
function usage(u) {
  if (!u) return h("div", { class: "sub" }, "usage unknown");
  const cost = u.costUSD ? `, ${u.costUSD.toFixed(2)} USD reported` : "";
  return h("div", { class: "sub" }, `${u.providerTurns} provider steps${cost}`);
}

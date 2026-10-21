// The Runs area: the runs that run now and the history of every recorded run
// of the observed state directories with its outcome, publication,
// organization, source and reported usage. A run opens in the inspector. A
// card above them starts and stops the one run this cockpit runs, when it is
// configured to; runs started elsewhere are only observed.

import { h, clear, when, duration } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";
import { runControlCard } from "./runcontrol.js";
import { outcomeOfSummary, buildText } from "./runfacts.js";

export function renderRuns(content) {
  const now = h("section", { class: "panel", "aria-labelledby": "runs-now" });
  const history = h("section", { class: "panel", "aria-labelledby": "runs-history" });
  const states = h("p", { class: "note", style: { margin: 0 } });
  const page = h("div", { class: "page" },
    h("div", { class: "pagehead" }, h("div", {},
      h("h2", {}, "Runs"),
      h("p", { class: "lead" }, "Start a run here when the cockpit has a run configuration and a Runtime profile, or start an Apply with markitect apply; it appears below while it runs. Stop ends only the run started here, incomplete with its evidence. A run started elsewhere is stopped where it was started, and pausing a replay never reaches a run."))),
    runControlCard(), now, history, states);
  content.append(page);
  const draw = (s) => {
    if (!page.isConnected) return unsubscribe();
    const live = s.liveRuns;
    clear(now).append(h("header", {}, h("h3", { id: "runs-now" }, "Running now")));
    now.append(live.length
      ? h("ul", { class: "rows" }, live.map((r) => h("li", {},
        h("a", { class: "row", href: `#/run/${encodeURIComponent(r.id)}` },
          h("span", { class: "lamp st-working" }),
          h("span", { class: "title" }, r.id),
          h("span", { class: "meta" }, r.health.run),
          h("span", { class: "sub" }, (r.account || [])[0]?.map((seg) => seg.text).join("") || `${r.records} records so far`)))))
      : h("p", { class: "void", style: { paddingTop: "4px" } }, "No run holds its state directory now."));
    clear(history).append(h("header", {}, h("h3", { id: "runs-history" }, "History")));
    if (s.errors.runs) {
      history.append(h("p", { class: "void", role: "alert" }, `The runs are not readable: ${s.errors.runs.message}`));
      return;
    }
    const runs = s.data.runs || [];
    history.append(runs.length ? runRows(runs) : h("p", { class: "void" }, h("strong", {}, "No run evidence"), "A run started here or with markitect apply appears once it holds a state directory this cockpit observes (the runtime directory's state/ for a project that is set up, or --state DIR)."));
    const dirs = s.data.project?.states || [];
    clear(states).append("State directories observed: ", ...dirs.map((d, i) => [i ? ", " : "", h("span", { class: "digest", title: d }, shortPath(d))]).flat());
  };
  const unsubscribe = store.subscribe(draw);
  if (Object.keys(store.data).length) draw(store);
  else refreshState();
}

// runRows is the run table.
export function runRows(runs) {
  const rows = runs.map((r) => {
    if (r.error && !r.id) return h("tr", {}, h("td", { colspan: 6 }, `${r.state}: ${r.error}`));
    const run = r.health?.run;
    const out = outcomeOfSummary(r, run);
    const outcome = h("span", { class: `outcome ${out.cls}` }, h("span", { class: "lamp" }), out.text);
    const span = r.started && r.last ? duration(new Date(r.last) - new Date(r.started)) : "";
    return h("tr", {},
      h("td", {}, h("a", { href: `#/run/${encodeURIComponent(r.id)}` }, r.id), h("div", { class: "sub", title: r.state }, shortPath(r.state))),
      h("td", {}, when(r.started), span && h("div", { class: "sub" }, span)),
      h("td", {}, outcome, r.publication && r.runStatus !== "verified" && h("div", { class: "sub" }, `publication ${r.publication}`), r.reason && h("div", { class: "sub" }, r.reason.length > 140 ? `${r.reason.slice(0, 139)}…` : r.reason)),
      h("td", {}, r.strategy ? h("strong", {}, r.strategy) : "—", r.build && h("div", { class: "sub", title: r.build }, buildText(r.build))),
      h("td", {}, `${r.managers} Manager${r.managers === 1 ? "" : "s"}`, h("div", { class: "sub" }, `${r.passes} Pass${r.passes === 1 ? "" : "es"}`), r.head && h("div", { class: "sub digest" }, `source ${r.head.slice(0, 12)}`)),
      h("td", {}, `${r.records} records`, usage(r.usage), r.error && h("div", { class: "st-fail" }, r.error)));
  });
  return h("table", { class: "table" },
    h("thead", {}, h("tr", {}, h("th", {}, "Run"), h("th", {}, "Started"), h("th", {}, "Outcome"), h("th", {}, "Strategy"), h("th", {}, "Organization and source"), h("th", {}, "Record"))),
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

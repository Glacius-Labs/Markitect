// The proposals inbox: every proposal of this cockpit on the left, the chosen
// one on the right with its check, difference and the owner's decision.

import { h, clear, when } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";
import { mountProposal, stateWords, stateClass } from "./proposal.js";

let filter = "open"; // remembered while the cockpit page lives

export function renderProposals(content, selected) {
  const listPane = h("div", { class: "pane" });
  const detail = h("div", { class: "pane" });
  content.append(h("div", { class: "master" }, listPane, detail));
  const listBody = h("div", {});
  const tabs = h("div", { class: "seg", role: "group", "aria-label": "Show" });
  const head = h("div", { class: "panehead" }, h("h2", {}, "Proposals"), h("div", { class: "actions" }, tabs));
  listPane.append(head, listBody);
  const drawTabs = () => {
    clear(tabs).append(...[["open", "Open"], ["all", "All"]].map(([key, text]) => {
      const b = h("button", { type: "button", "aria-pressed": String(filter === key) }, text);
      b.addEventListener("click", () => { filter = key; drawTabs(); drawList(store); });
      return b;
    }));
  };
  const drawList = (s) => {
    clear(listBody);
    if (s.errors.proposals) {
      listBody.append(h("p", { class: "void" }, h("strong", {}, "Not available"), "Proposals need the cockpit to be started with --repo."));
      return;
    }
    const all = s.data.proposals || [];
    const shown = filter === "open" ? all.filter((p) => p.state === "proposed" || p.state === "publishing") : all;
    if (!shown.length) {
      listBody.append(h("p", { class: "void" }, h("strong", {}, filter === "open" ? "Nothing waits for a decision" : "No proposal yet"),
        "Proposals come from the ", h("a", { href: "#/settings" }, "settings form"), " and from ", h("a", { href: "#/session" }, "model sessions"), ". They live as long as this cockpit."));
      return;
    }
    listBody.append(h("ul", { class: "rows" }, shown.map((p) => h("li", {},
      h("a", { class: "row", href: `#/proposal/${encodeURIComponent(p.id)}`, "aria-current": String(p.id === selected) },
        h("span", { class: `lamp ${stateClass[p.state] || ""}` }),
        h("span", { class: "title" }, p.title),
        h("span", { class: "meta" }, p.kind),
        h("span", { class: "sub" }, `${p.id}, ${stateWords[p.state] || p.state}, check ${p.check.status}, made ${when(p.created)}`))))));
  };
  drawTabs();
  const unsubscribe = store.subscribe((s) => {
    if (!listPane.isConnected) return unsubscribe();
    drawList(s);
  });
  if (Object.keys(store.data).length) drawList(store);
  else refreshState();

  if (!selected) {
    detail.append(h("div", { class: "panebody" }, h("p", { class: "void" }, h("strong", {}, "Choose a proposal"), "Its check, its difference and your decision appear here.")));
    return;
  }
  const body = h("div", { class: "panebody" });
  detail.append(h("div", { class: "panehead" }, h("h2", {}, selected)), body);
  mountProposal(body, selected);
}

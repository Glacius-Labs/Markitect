// Decisions: what waits for the owner. The proposals this cockpit holds in one
// list; the chosen one beside it with its check, difference and decision.
// Without a choice, the first waiting proposal opens, as in a mail inbox. A
// run asks the owner nothing: no question crosses Root during a Run.

import { h, clear, when, empty } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";
import { mountProposal, stateWords } from "./proposal.js";

let filter = "waiting"; // remembered while the cockpit page lives

export function renderDecisions(content, selected) {
  const listPane = h("div", { class: "pane" });
  const detail = h("div", { class: "pane" });
  content.append(h("div", { class: "master" }, listPane, detail));
  const tabs = h("div", { class: "seg", role: "group", "aria-label": "Show" });
  const title = h("h2", {});
  const listBody = h("div", {});
  listPane.append(h("div", { class: "panehead" }, title, h("div", { class: "actions" }, tabs)), listBody);
  // The filter buttons stay and change state, so focus stays on them.
  const buttons = [["waiting", "Waiting"], ["all", "All"]].map(([key, text]) => {
    const b = h("button", { type: "button", dataset: { key } }, text);
    b.addEventListener("click", () => { filter = key; drawTabs(); drawList(store); });
    return b;
  });
  tabs.append(...buttons);
  const drawTabs = () => {
    title.textContent = filter === "waiting" ? "Waiting for you" : "All decisions";
    for (const b of buttons) b.setAttribute("aria-pressed", String(b.dataset.key === filter));
  };
  // The list is drawn again only when what it shows changes, so a poll keeps
  // the focus on a row.
  let shownList = null;
  const drawList = (s) => {
    const shown = items(s).filter((x) => filter === "all" || x.open);
    const first = shown.find((x) => x.open);
    if (!selected && first) {
      location.replace(proposalHref(first.p));
      return;
    }
    const sig = JSON.stringify([filter, shown, s.errors.proposals?.message]);
    if (sig === shownList) return;
    shownList = sig;
    clear(listBody);
    if (!shown.length) {
      listBody.append(empty(filter === "waiting" ? "Nothing waits for you" : "No proposal yet",
        "They come from ", h("a", { href: "#/settings" }, "Settings"), " and ", h("a", { href: "#/session" }, "model sessions"), "."));
    } else {
      listBody.append(h("ul", { class: "rows" }, shown.map((x) => h("li", {}, proposalRow(x.p, selected)))));
    }
    if (s.errors.proposals) listBody.append(h("p", { class: "note pad" }, "Proposals need the cockpit to be started with --repo."));
  };
  drawTabs();
  detailView(detail, selected);
  const unsubscribe = store.subscribe((s) => {
    if (!listPane.isConnected) return unsubscribe();
    drawList(s);
  });
  if (Object.keys(store.data).length) {
    drawList(store);
  } else {
    refreshState();
  }
}

// items lists the proposals; open ones are those that wait for the owner or
// are being written.
function items(s) {
  const proposals = (s.data.proposals || []).map((p) => ({ open: p.state === "proposed" || p.state === "publishing", p }));
  return [...proposals.filter((x) => x.open), ...proposals.filter((x) => !x.open)];
}

function proposalRow(p, selected) {
  const current = selected?.kind === "proposal" && selected.id === p.id;
  return h("a", { class: "row", href: proposalHref(p), "aria-current": String(current) },
    h("span", { class: "dkind proposal" }, "Proposal"),
    h("span", { class: "title" }, p.title),
    h("span", { class: `meta outcome ${p.check.status === "passed" ? "st-done" : "st-fail"}` }, h("span", { class: "lamp" }), `check ${p.check.status}`),
    h("span", { class: "sub" }, `${p.id}, ${stateWords[p.state] || p.state}, made ${when(p.created)}`));
}

// detailView draws the chosen proposal into the right pane.
function detailView(pane, selected) {
  if (!selected) {
    pane.append(h("div", { class: "panebody blueprint fillpane" }, empty("Choose a proposal", "It shows its check, difference and your decision.")));
    return;
  }
  const body = h("div", { class: "panebody" });
  pane.append(h("div", { class: "panehead" }, h("h2", {}, "Proposal"), h("span", { class: "coord" }, selected.id)), body);
  mountProposal(body, selected.id);
}

function proposalHref(p) {
  return `#/proposal/${encodeURIComponent(p.id)}`;
}

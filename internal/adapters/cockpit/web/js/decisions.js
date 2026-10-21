// Decisions: what waits for the owner. The questions runs ask the owner and
// this cockpit's proposals in one list; the chosen one beside it — a question
// with its answer form, a proposal with its check, difference and decision.
// Without a choice, the first waiting item opens, as in a mail inbox.

import { h, clear, when } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";
import { mountProposal, stateWords } from "./proposal.js";
import { answerForm, answerInProgress } from "./answer.js";

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
      location.replace(first.kind === "question" ? questionHref(first.q) : proposalHref(first.p));
      return;
    }
    const sig = JSON.stringify([filter, shown, s.errors.questions?.message, s.errors.proposals?.message]);
    if (sig === shownList) return;
    shownList = sig;
    clear(listBody);
    if (!shown.length) {
      listBody.append(h("p", { class: "void" }, h("strong", {}, filter === "waiting" ? "Nothing waits for you" : "No question or proposal yet"),
        "A run asks you here when its model leaves something open. Proposals come from the ", h("a", { href: "#/settings" }, "settings"),
        " and from ", h("a", { href: "#/session" }, "model sessions"), "; they live as long as this cockpit."));
    } else {
      listBody.append(h("ul", { class: "rows" }, shown.map((x) => h("li", {}, x.kind === "question" ? questionRow(x.q, selected) : proposalRow(x.p, selected)))));
    }
    for (const [part, text] of [["questions", "The owner questions are not readable"], ["proposals", "Proposals need the cockpit to be started with --repo"]]) {
      if (s.errors[part]) listBody.append(h("p", { class: "note", style: { padding: "0 18px" } }, `${text}.`));
    }
  };
  drawTabs();
  const drawDetail = detailView(detail, selected);
  const unsubscribe = store.subscribe((s) => {
    if (!listPane.isConnected) return unsubscribe();
    drawList(s);
    drawDetail(s);
  });
  if (Object.keys(store.data).length) {
    drawList(store);
    drawDetail(store);
  } else {
    refreshState();
  }
}

// items puts the questions and the proposals into one list; open ones are
// those that wait for the owner or are being written.
function items(s) {
  const questions = (s.data.questions || []).map((q) => ({ kind: "question", open: q.answerable, q }));
  const proposals = (s.data.proposals || []).map((p) => ({ kind: "proposal", open: p.state === "proposed" || p.state === "publishing", p }));
  return [...questions.filter((x) => x.open), ...proposals.filter((x) => x.open), ...questions.filter((x) => !x.open), ...proposals.filter((x) => !x.open)];
}

function questionRow(q, selected) {
  const current = selected?.kind === "question" && selected.run === q.run && selected.id === q.id;
  return h("a", { class: "row", href: questionHref(q), "aria-current": String(current) },
    h("span", { class: "dkind question" }, "Question"),
    h("span", { class: "title" }, firstLine(q.text)),
    h("span", { class: `meta outcome ${q.answerable ? "st-owner" : "st-none"}` }, h("span", { class: "lamp" }), q.answerable ? "waits" : q.status),
    h("span", { class: "sub" }, `${q.id} of run ${q.run}, asked ${when(q.asked)}`));
}

function proposalRow(p, selected) {
  const current = selected?.kind === "proposal" && selected.id === p.id;
  return h("a", { class: "row", href: proposalHref(p), "aria-current": String(current) },
    h("span", { class: "dkind proposal" }, "Proposal"),
    h("span", { class: "title" }, p.title),
    h("span", { class: `meta outcome ${p.check.status === "passed" ? "st-done" : "st-fail"}` }, h("span", { class: "lamp" }), `check ${p.check.status}`),
    h("span", { class: "sub" }, `${p.id}, ${stateWords[p.state] || p.state}, made ${when(p.created)}`));
}

// detailView draws the chosen item into the right pane and returns the
// function that keeps it current with the shared observation.
function detailView(pane, selected) {
  if (!selected) {
    pane.append(h("div", { class: "panebody blueprint", style: { minHeight: "100%" } },
      h("p", { class: "void" }, h("strong", {}, "Choose a question or a proposal"), "A question shows the run's text and your answer; a proposal its check, difference and your decision.")));
    return () => {};
  }
  const body = h("div", { class: "panebody" });
  if (selected.kind === "proposal") {
    pane.append(h("div", { class: "panehead" }, h("h2", {}, "Proposal"), h("span", { class: "coord" }, selected.id)), body);
    mountProposal(body, selected.id);
    return () => {};
  }
  const head = h("div", { class: "panehead" }, h("h2", {}, `Question ${selected.id}`),
    h("div", { class: "actions" }, h("a", { class: "btn small", href: `#/run/${encodeURIComponent(selected.run)}?sel=${encodeURIComponent(`question:${selected.id}`)}` }, "Open the run here")));
  pane.append(head, body);
  body.append(h("p", { class: "note", role: "status" }, "Reading the question…"));
  let shown = null;
  return (s) => {
    const q = (s.data.questions || []).find((x) => x.run === selected.run && x.id === selected.id);
    const key = q ? `${q.status}:${q.answerable}:${Boolean(s.data.project?.answers)}` : s.data.questions ? "missing" : "reading";
    if (key === shown || answerInProgress(body)) return;
    shown = key;
    if (!q) {
      if (s.data.questions || s.errors.questions) clear(body).append(h("p", { class: "refusal", role: "alert" }, `The observed runs hold no question ${selected.id} of run ${selected.run}.`));
      return;
    }
    clear(body).append(questionView(q, Boolean(s.data.project?.answers)));
  };
}

function questionView(q, answers) {
  return h("div", { class: "stack" },
    h("div", { class: "block" },
      h("p", { class: `outcome big ${q.answerable ? "st-owner" : "st-none"}`, style: { margin: 0 } }, h("span", { class: "lamp" }), q.answerable ? "Waits for your answer" : cap(q.status)),
      h("dl", { class: "facts" },
        h("dt", {}, "Run"), h("dd", {}, h("a", { href: `#/run/${encodeURIComponent(q.run)}` }, q.run)),
        h("dt", {}, "Asked"), h("dd", {}, when(q.asked)),
        h("dt", {}, "State"), h("dd", {}, `${cap(q.why)}.`))),
    h("div", { class: "block" }, h("h3", {}, "Root's question"), h("div", { class: "prose quote agent" }, q.text)),
    q.answerable && answers ? h("div", { class: "decide" }, answerForm(q.run, q.id)) : null,
    q.answerable && !answers ? h("p", { class: "note", style: { margin: 0 } }, "This cockpit writes no answers; answer with markitect answer.") : null);
}

function questionHref(q) {
  return `#/question/${encodeURIComponent(q.run)}/${encodeURIComponent(q.id)}`;
}

function proposalHref(p) {
  return `#/proposal/${encodeURIComponent(p.id)}`;
}

function firstLine(text) {
  const line = (text || "").trim().split("\n")[0];
  return line.length > 120 ? `${line.slice(0, 119)}…` : line;
}

function cap(text) {
  return text ? text[0].toUpperCase() + text.slice(1) : "";
}

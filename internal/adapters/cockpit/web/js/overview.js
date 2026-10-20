// The overview: what needs the owner now and what the project is. The
// project's observed state, the open model session, the proposals waiting for
// a decision, the runs that run now with the owner questions they wait for,
// and the latest runs. Only answering a question writes anything here.

import { api } from "./api.js";
import { h, clear, when, shortDigest } from "./dom.js";
import { store } from "./store.js";
import { answerForm, answerInProgress } from "./answer.js";
import { refreshState } from "./shell.js";
import { stateWords as proposalWords, stateClass as proposalClass } from "./proposal.js";
import { runRows } from "./runs.js";

export function renderOverview(content) {
  const page = h("div", { class: "page" });
  content.append(page);
  const project = h("section", { class: "panel", "aria-labelledby": "ov-project" });
  const session = h("section", { class: "panel", "aria-labelledby": "ov-session" });
  const proposals = h("section", { class: "panel", "aria-labelledby": "ov-proposals" });
  const now = h("section", { class: "panel", "aria-labelledby": "ov-now" });
  const questions = h("section", { class: "panel", "aria-labelledby": "ov-questions" });
  const recent = h("section", { class: "panel wide", "aria-labelledby": "ov-runs" });
  page.append(h("div", { class: "grid" }, project, session, proposals, now, questions, recent));
  const draw = (s) => {
    if (!page.isConnected) return unsubscribe();
    drawProject(project, s);
    drawSession(session, s);
    drawProposals(proposals, s);
    drawNow(now, s);
    drawQuestions(questions, s);
    drawRecent(recent, s);
  };
  const unsubscribe = store.subscribe(draw);
  if (Object.keys(store.data).length) draw(store);
  else refreshState();
}

function head(id, title, more) {
  return h("header", {}, h("h3", { id }, title), more || null);
}

function notReadable(section, id, title, err) {
  clear(section).append(head(id, title), h("p", { class: "empty", role: "alert" }, `Not readable: ${err.message}`));
}

function drawProject(section, s) {
  const part = s.data.project;
  if (s.errors.project) return notReadable(section, "ov-project", "Project", s.errors.project);
  const p = part?.project;
  const observe = h("button", { class: "btn small", type: "button", disabled: part?.observing }, part?.observing ? "Observing…" : "Observe again");
  observe.addEventListener("click", async () => {
    observe.disabled = true;
    observe.textContent = "Observing…";
    try { await api.observeProject(); } catch { /* the panel shows the next state */ }
    refreshState();
  });
  clear(section).append(head("ov-project", p?.modelID ? `Project ${p.modelID}` : "Project", part?.configured ? observe : null));
  const body = h("div", { class: "body" });
  section.append(body);
  if (!part?.configured) {
    body.append(h("p", { class: "empty" }, h("strong", {}, "No repository"), "The cockpit was started without --repo; only the runs of the observed state directories are shown."));
    return;
  }
  if (!p) {
    body.append(h("p", { class: "empty", role: "status" }, h("strong", {}, "Observing the repository"), "The structural preparation of the working tree runs; on a large model this takes a while."));
    return;
  }
  const ok = p.status === "passed";
  body.append(
    h("div", { class: "stat" },
      h("div", {}, h("b", {}, String(p.packages)), h("span", {}, "Packages")),
      h("div", {}, h("b", {}, String(p.definitions)), h("span", {}, "Definitions")),
      h("div", {}, h("b", { class: ok ? "st-done" : "st-fail" }, ok ? "passed" : p.status || "not completed"), h("span", {}, "Structural check"))),
    h("dl", { class: "facts", style: { marginTop: "14px" } },
      p.modelDigest ? [h("dt", {}, "Canonical model"), h("dd", {}, h("span", { class: "digest", title: p.modelDigest }, shortDigest(p.modelDigest)))] : null,
      p.head ? [h("dt", {}, "Source head"), h("dd", {}, h("span", { class: "digest" }, p.head.slice(0, 12)))] : null,
      h("dt", {}, "Observed"), h("dd", {}, `${when(p.observedAt)}, from the working tree`),
      p.error ? [h("dt", {}, "Problem"), h("dd", { class: "st-fail" }, p.error)] : null),
    h("p", { class: "note", style: { margin: "12px 0 0" } }, h("a", { href: "#/model" }, "Browse the model"), " or ", h("a", { href: "#/settings" }, "change the execution settings"), "."));
}

function drawSession(section, s) {
  const open = s.openSession;
  clear(section).append(head("ov-session", "Model session", h("a", { class: "more", href: "#/session" }, open ? "Open" : "Start")));
  const body = h("div", { class: "body" });
  section.append(body);
  if (s.errors.sessions || !s.sessionsAvailable) {
    body.append(h("p", { class: "empty" }, h("strong", {}, "Not available"), "Model sessions need the cockpit to be started with --run-config and --runtime-profile."));
    return;
  }
  if (!open) {
    body.append(h("p", { class: "empty" }, h("strong", {}, "No open session"), "Change the model by prompt: an Agent works on its own copy and proposes; you decide."));
    return;
  }
  const last = open.turns[open.turns.length - 1];
  const cls = open.state === "working" || open.state === "preparing" ? "st-working" : open.state === "halted" ? "st-fail" : "st-waiting";
  body.append(
    h("p", { class: `outcome ${cls}`, style: { margin: "0 0 8px" } }, h("span", { class: "lamp" }), h("strong", {}, open.state)),
    last ? h("p", { class: "prose", style: { margin: "0 0 8px" } }, last.prompt.length > 220 ? `${last.prompt.slice(0, 219)}…` : last.prompt) : null,
    h("p", { class: "note", style: { margin: 0 } }, `${open.turns.length} of ${open.maxTurns || "?"} Agent turns; profile ${open.profile || "not yet known"}.`));
}

function drawProposals(section, s) {
  const open = s.openProposals;
  clear(section).append(head("ov-proposals", "Proposals", h("a", { class: "more", href: "#/proposals" }, "All proposals")));
  if (s.errors.proposals) {
    section.append(h("p", { class: "empty" }, h("strong", {}, "Not available"), "Proposals need the cockpit to be started with --repo."));
    return;
  }
  if (!open.length) {
    section.append(h("p", { class: "empty" }, h("strong", {}, "Nothing waits for a decision"), "Proposals come from the settings form and from model sessions."));
    return;
  }
  section.append(h("ul", { class: "rows" }, open.slice(0, 6).map((p) => h("li", {},
    h("a", { class: "row", href: `#/proposal/${encodeURIComponent(p.id)}` },
      h("span", { class: `lamp ${proposalClass[p.state] || ""}` }),
      h("span", { class: "title" }, p.title),
      h("span", { class: "meta" }, p.check.status === "passed" ? "check passed" : `check ${p.check.status}`),
      h("span", { class: "sub" }, `${p.id}, ${proposalWords[p.state] || p.state}, made ${when(p.created)}`))))));
}

function drawNow(section, s) {
  clear(section).append(head("ov-now", "Running now", h("a", { class: "more", href: "#/runs" }, "All runs")));
  if (s.errors.runs) return section.append(h("p", { class: "empty", role: "alert" }, `The runs are not readable: ${s.errors.runs.message}`));
  const live = s.liveRuns;
  if (!live.length) {
    section.append(h("p", { class: "empty" }, h("strong", {}, "No run holds its state directory"), "A run started with markitect apply appears here while it runs. The cockpit never starts or stops one."));
    return;
  }
  section.append(h("ul", { class: "rows" }, live.map((r) => h("li", {},
    h("a", { class: "row", href: `#/run/${encodeURIComponent(r.id)}` },
      h("span", { class: "lamp st-working" }),
      h("span", { class: "title" }, r.id),
      h("span", { class: "meta" }, r.health.run),
      h("span", { class: "sub" }, (r.account || [])[0]?.map((seg) => seg.text).join("") || `${r.records} records so far`))))));
}

function drawQuestions(section, s) {
  if (answerInProgress(section)) return;
  clear(section).append(head("ov-questions", "Owner questions"));
  if (s.errors.questions) return section.append(h("p", { class: "empty", role: "alert" }, `The questions are not readable: ${s.errors.questions.message}`));
  const questions = s.data.questions || [];
  const answers = Boolean(s.data.project?.answers);
  if (!questions.length) {
    section.append(h("p", { class: "empty" }, h("strong", {}, "No question"), "No run in the observed state directories has asked the owner a question."));
    return;
  }
  const waiting = questions.filter((q) => q.answerable);
  const shown = waiting.length ? waiting : questions.slice(0, 3);
  for (const q of shown) {
    const text = h("div", { class: "prose quote agent" }, q.text);
    const card = h("article", { class: `qcard${q.answerable ? " waiting" : ""}` },
      h("h4", {}, `${q.id} of run `, h("a", { href: `#/run/${encodeURIComponent(q.run)}?sel=${encodeURIComponent(`question:${q.id}`)}` }, q.run)),
      h("p", { class: "note", style: { margin: "0 0 6px" } }, `Asked ${when(q.asked)}; ${q.status}. ${q.why.charAt(0).toUpperCase()}${q.why.slice(1)}.`),
      q.answerable ? text : h("details", {}, h("summary", {}, "Question text"), text));
    if (q.answerable && answers) card.append(answerForm(q.run, q.id));
    section.append(card);
  }
  if (!waiting.length) section.append(h("p", { class: "note", style: { padding: "0 14px 10px", margin: 0 } }, "No question waits for an answer now; the latest are shown."));
}

function drawRecent(section, s) {
  clear(section).append(head("ov-runs", "Recent runs", h("a", { class: "more", href: "#/runs" }, "All runs")));
  if (s.errors.runs) return;
  const runs = (s.data.runs || []).slice(0, 6);
  if (!runs.length) {
    section.append(h("p", { class: "empty" }, h("strong", {}, "No run evidence"), "Start the cockpit with the state directory of an Apply (--state DIR or --run-config PATH)."));
    return;
  }
  section.append(h("div", { class: "body flush" }, runRows(runs)));
}


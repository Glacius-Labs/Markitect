// The cockpit's entry: the project and its bound identities, the runs that
// run now, the owner questions they wait for, the run history with outcomes,
// the proposals for the project's settings, and which actions the cockpit
// offers. Only answering and accepting a proposal write anything.

import { api } from "./api.js";
import { h, clear, when, duration, shortDigest } from "./dom.js";
import { answerForm, answerInProgress } from "./answer.js";
import { proposalList } from "./proposal.js";

const outcomeClass = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" };
const POLL_MS = 3000;

export function renderRuns(app) {
  document.title = "Markitect cockpit";
  clear(app);
  const main = h("main", { id: "main", class: "runs" });
  app.append(main);
  const sections = {
    project: h("section", { class: "band", "aria-labelledby": "h-project" }),
    now: h("section", { class: "band", "aria-labelledby": "h-now" }),
    questions: h("section", { class: "band", "aria-labelledby": "h-questions" }),
    proposals: h("section", { class: "band", "aria-labelledby": "h-proposals" }),
    runs: h("section", { class: "band", "aria-labelledby": "h-runs" }),
    actions: h("section", { class: "band", "aria-labelledby": "h-actions" }),
  };
  main.append(...Object.values(sections));
  let stopped = false;
  let answers = false;
  const settled = (promise) => promise.then((value) => ({ status: "fulfilled", value }), (reason) => ({ status: "rejected", reason }));
  // Each section renders as soon as its part arrives; the project's structural
  // check can take seconds and never holds up the runs.
  const refresh = (observe = false) => {
    if (stopped) return;
    const live = () => !stopped && main.isConnected;
    settled(observe ? api.observeProject() : api.project()).then((project) => {
      if (!live()) return;
      answers = Boolean(project.value?.answers);
      const authoring = Boolean(project.value?.proposals);
      renderProject(sections.project, project, () => refresh(true));
      renderActions(sections.actions, answers, authoring);
      if (authoring) settled(api.proposals()).then((proposals) => live() && renderProposals(sections.proposals, proposals));
      else clear(sections.proposals);
      settled(api.questions()).then((questions) => live() && renderQuestions(sections.questions, questions, answers));
    });
    settled(api.runs()).then((runs) => {
      if (!live()) return;
      renderNow(sections.now, runs);
      renderHistory(sections.runs, runs);
    });
  };
  refresh();
  const timer = setInterval(() => {
    if (!main.isConnected) {
      stopped = true;
      clearInterval(timer);
      return;
    }
    if (document.visibilityState === "visible") refresh();
  }, POLL_MS);
}

function failed(section, title, result) {
  clear(section).append(h("h2", {}, title), h("p", { role: "alert" }, `Not readable: ${result.reason?.message || "unknown error"}`));
}

function renderProject(section, result, onRefresh) {
  if (result.status !== "fulfilled") return failed(section, "Project", result);
  const { project, states, observing, configured } = result.value;
  const head = h("div", { class: "projecthead" }, h("h1", { id: "h-project" }, project?.modelID ? `Project ${project.modelID}` : "Markitect cockpit"));
  clear(section).append(head);
  if (!configured) {
    section.append(h("p", { class: "lead" }, "Started without a repository: only the runs in the observed state directories are shown."));
  } else if (!project) {
    section.append(h("p", { class: "lead", role: "status" }, "Observing the repository's model and source…"));
  } else {
    const ok = project.status === "passed";
    head.append(h("button", { class: "btn", type: "button", onclick: onRefresh, disabled: observing }, observing ? "Observing…" : "Observe again"));
    section.append(h("dl", { class: "facts" },
      h("dt", {}, "Repository"), h("dd", { class: "digest" }, project.repository),
      h("dt", {}, "Structural check"), h("dd", { class: ok ? "st-done" : "st-fail" }, ok ? "passed" : project.status || "not completed", project.error ? h("span", { class: "muted" }, ` (${project.error})`) : null),
      project.modelDigest ? [h("dt", {}, "Canonical model"), h("dd", {}, h("span", { class: "digest" }, shortDigest(project.modelDigest)), ` with ${project.packages} Package${project.packages === 1 ? "" : "s"}`)] : null,
      project.head ? [h("dt", {}, "Source head"), h("dd", { class: "digest" }, project.head.slice(0, 12))] : null,
      h("dt", {}, "Observed"), h("dd", {}, `${when(project.observedAt)} from the working tree`)));
  }
  section.append(h("p", { class: "note" }, `State directories observed: ${states.join(", ")}`));
}

function renderNow(section, result) {
  clear(section).append(h("h2", { id: "h-now" }, "Running now"));
  if (result.status !== "fulfilled") return section.append(h("p", { role: "alert" }, `The runs are not readable: ${result.reason.message}`));
  const live = result.value.filter((r) => r.health && (r.health.run === "running" || r.health.run === "finishing"));
  if (!live.length) {
    section.append(h("p", { class: "muted" }, "No observed run holds its state directory now. A run started with markitect apply appears here while it runs."));
    return;
  }
  for (const r of live) {
    section.append(h("article", { class: "nowcard" },
      h("h3", {}, h("a", { href: `#/run/${encodeURIComponent(r.id)}` }, r.id), " ", h("span", { class: "outcome st-done" }, h("span", { class: "lamp" }), r.health.run === "finishing" ? "finishing" : "running")),
      (r.account || []).map((sentence) => h("p", {}, sentence.map((seg) => seg.text).join(""))),
      h("p", { class: "note" }, `${r.records} records so far; ${r.health.why}.`)));
  }
}

// renderQuestions lists the owner questions. While the owner writes an
// answer, the section is not redrawn under the form.
function renderQuestions(section, result, answers) {
  if (answerInProgress(section)) return;
  clear(section).append(h("h2", { id: "h-questions" }, "Owner questions"));
  if (result.status !== "fulfilled") return section.append(h("p", { role: "alert" }, `The questions are not readable: ${result.reason.message}`));
  const questions = result.value;
  if (!questions.length) {
    section.append(h("p", { class: "muted" }, "No run in the observed state directories has asked the owner a question."));
    return;
  }
  const waiting = questions.filter((q) => q.answerable).length;
  section.append(h("p", { class: "muted" }, waiting ? `${waiting} question${waiting === 1 ? " waits" : "s wait"} for an answer.` : "No question waits for an answer now."));
  for (const q of questions) {
    // A question nobody can answer any more keeps its text one click away.
    const text = h("div", { class: "prose quote agent" }, q.text);
    const card = h("article", { class: `qcard${q.answerable ? " waiting" : ""}` },
      h("h3", {}, `${q.id} of run `, h("a", { href: `#/run/${encodeURIComponent(q.run)}?sel=${encodeURIComponent(`question:${q.id}`)}` }, q.run)),
      h("p", { class: "muted" }, `Asked ${when(q.asked)}; ${q.status}. ${q.why.charAt(0).toUpperCase()}${q.why.slice(1)}.`),
      q.answerable ? text : h("details", {}, h("summary", {}, "Question text"), text));
    if (q.answerable && answers) card.append(answerForm(q.run, q.id));
    section.append(card);
  }
}

function renderProposals(section, result) {
  clear(section).append(h("div", { class: "projecthead" }, h("h2", { id: "h-proposals" }, "Settings and proposals"),
    h("a", { class: "btn", href: "#/settings" }, "Change the execution settings")));
  if (result.status !== "fulfilled") return section.append(h("p", { role: "alert" }, `The proposals are not readable: ${result.reason.message}`));
  section.append(proposalList(result.value));
}

function renderHistory(section, result) {
  clear(section).append(h("h2", { id: "h-runs" }, "Runs and outcomes"));
  if (result.status !== "fulfilled") return section.append(h("p", { role: "alert" }, `The runs are not readable: ${result.reason.message}`));
  const runs = result.value;
  if (!runs.length) {
    section.append(h("p", {}, "No run evidence was found. Start the cockpit with the state directory of an Apply (--state DIR or --run-config PATH)."));
    return;
  }
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
      h("td", {}, h("a", { href: `#/run/${encodeURIComponent(r.id)}` }, r.id), h("div", { class: "state" }, r.state)),
      h("td", {}, when(r.started), span && h("div", { class: "faint" }, span)),
      h("td", {}, outcome, r.publication && h("div", { class: "faint" }, `publication ${r.publication}`), r.reason && h("div", { class: "faint" }, r.reason.length > 140 ? `${r.reason.slice(0, 139)}…` : r.reason)),
      h("td", {}, `${r.managers} Manager${r.managers === 1 ? "" : "s"}`, h("div", { class: "faint" }, `${r.passes} Pass${r.passes === 1 ? "" : "es"}`), r.head && h("div", { class: "faint digest" }, `source ${r.head.slice(0, 12)}`)),
      h("td", {}, `${r.records} records`, usage(r.usage), r.error && h("div", { class: "st-fail" }, r.error)));
  });
  section.append(h("table", { class: "runtable" },
    h("thead", {}, h("tr", {}, h("th", {}, "Run"), h("th", {}, "Started"), h("th", {}, "Outcome"), h("th", {}, "Organization and source"), h("th", {}, "Record"))),
    h("tbody", {}, rows)));
}

// usage states what the report says about provider use; without a report it
// is unknown, not zero.
function usage(u) {
  if (!u) return h("div", { class: "faint" }, "usage unknown");
  const cost = u.costUSD ? `, ${u.costUSD.toFixed(2)} USD reported` : "";
  return h("div", { class: "faint" }, `${u.providerTurns} provider steps${cost}`);
}

function renderActions(section, answers, authoring) {
  clear(section).append(h("h2", { id: "h-actions" }, "What the cockpit can do"),
    h("dl", { class: "facts" },
      h("dt", {}, "Answer an owner question"),
      h("dd", {}, answers ? "Available above while a running run waits for the answer. It writes through the same owner channel as markitect answer; the run records the answer when it reads it." : "Not configured in this cockpit."),
      h("dt", {}, "Change the execution settings"),
      h("dd", {}, authoring ? "Budget, role profiles, per-Package profiles and team, as a proposal: the structural check judges it, you review the difference, and only your acceptance writes .markitect/execution.yaml. Nothing is committed and no Apply starts." : "Needs the cockpit to be started with --repo PATH."),
      h("dt", {}, "Start an Apply"),
      h("dd", {}, "Not offered. Start it with markitect apply --run-config PATH --runtime-profile PATH; it appears here while it runs. Starting from the cockpit would start provider work; whether it should is an owner decision that has not been made."),
      h("dt", {}, "Stop a run"),
      h("dd", {}, "No stop command exists. Interrupting the markitect apply process ends the run incomplete with its evidence. Pausing a replay or a live view never stops a run.")));
}

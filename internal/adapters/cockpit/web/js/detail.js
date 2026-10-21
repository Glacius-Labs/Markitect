// The detail plate: what the record says about the selected Manager, Turn,
// proposal, finding, question, message, Pass or check, up to the boundary.
// Text an Agent wrote is set as prose; everything else is the Orchestrator's
// record. Nothing here infers a delivery, a reading or a success.

import { h, clear, clock, since, label, stateWord, shortDigest, duration } from "./dom.js";
import { activity } from "./diagram.js";
import { api } from "./api.js";

export function renderDetail(aside, ctx) {
  const token = (aside._token = (aside._token || 0) + 1);
  clear(aside);
  const close = h("button", { class: "btn icon close", "aria-label": "Close details", onclick: () => ctx.select(null) }, "×");
  const body = h("div", {});
  aside.append(close, body);
  const ref = ctx.ref || "run";
  const [kind, ...rest] = ref.split(":");
  const id = rest.join(":");
  const r = (target, text) => target ? h("button", { class: "ref", onclick: () => ctx.select(target) }, text) : text;
  const later = (records) => token === aside._token && records;
  const load = async () => later((await ctx.detail(ref)).records);
  const c = { ...ctx, r, body, load, id };
  switch (kind) {
    case "manager": manager(c); break;
    case "proposal": proposal(c); break;
    case "finding": finding(c); break;
    case "question": question(c); break;
    case "turn": turn(c); break;
    case "pass": pass(c); break;
    case "check": check(c); break;
    case "problem": problem(c); break;
    case "seq": record(c); break;
    case "owner": owner(c); break;
    case "orchestrator": orchestrator(c); break;
    default: overview(c);
  }
}

function title(c, text, kicker) {
  c.body.append(h("h2", {}, text));
  if (kicker) c.body.append(h("p", { class: "kicker" }, kicker));
}

function section(c, name, ...children) {
  c.body.append(h("h3", {}, name), ...children.filter((x) => x !== null && x !== undefined && x !== false));
}

function dl(pairs) {
  return h("dl", {}, pairs.filter(Boolean).flatMap(([k, v]) => [h("dt", {}, k), h("dd", {}, v)]));
}

function prose(text, cls = "") {
  return h("div", { class: `prose quote agent ${cls}` }, text || "");
}

function pending(c, what = "Reading the record…") {
  const p = h("p", { class: "note" }, what);
  c.body.append(p);
  return p;
}

function failed(p, err) {
  p.textContent = `The record could not be read: ${err.message}`;
}

function verdictTag(v) {
  if (!v) return null;
  return h("span", { class: `tag ${v === "satisfied" ? "ok" : "fail"}` }, v.replace("_", " "));
}

// ---------- Manager ----------

function manager(c) {
  const { view, id, r } = c;
  const m = view.managers.find((x) => x.path === id);
  if (!m) {
    title(c, label(id), "Not part of the recorded organization at this boundary.");
    return;
  }
  const pv = view.passes.find((p) => p.id === view.current);
  title(c, label(m.path), m.path === "." ? "Root Manager: coordinates the whole model" : `Manager of Package ${m.path}`);
  c.body.append(h("p", { class: `st-${m.state || "none"}`, style: { display: "flex", gap: "8px", alignItems: "center", margin: "0" } },
    h("span", { class: "lamp" }), h("strong", {}, stateWord(m.state)), h("span", { class: "muted" }, pv ? `in ${pv.purpose} Pass ${pv.id}` : "")));
  if (m.wait) c.body.append(why(c, m));
  const act = activity(m);
  if (m.turn) {
    section(c, "Active turn", h("p", {}, r(`turn:${m.turn.id}`, `${m.turn.role} turn ${m.turn.short}`), `, ${m.turn.readOnly ? "read-only" : "may write its own areas"}, running for ${since(m.turn.started, view.boundary.at)}`),
      m.turn.role === "reviewer" ? h("p", { class: "note" }, "A reviewer turn assesses the immutable candidate; it does not implement.") : null);
  } else if (act && !m.wait) {
    c.body.append(h("p", { class: "muted" }, act));
  }
  if (m.helpers?.length) section(c, "Helpers", h("ul", {}, m.helpers.map((x) => h("li", {}, `${x.role} ${x.id}`))));
  const mine = view.proposals.filter((p) => p.manager === m.path && p.pass === view.current);
  section(c, "Submissions in this Pass", mine.length
    ? h("table", {}, h("tbody", {}, mine.map((p) => h("tr", {}, h("td", {}, r(`proposal:${p.id}`, p.id)), h("td", {}, p.kind), h("td", {}, statusText(p))))))
    : h("p", { class: "empty" }, "None recorded yet."));
  const own = view.findings.filter((f) => f.owner === m.path);
  if (own.length) {
    section(c, "Findings it owns", h("table", {}, h("tbody", {}, own.map((f) => h("tr", {},
      h("td", {}, r(`finding:${f.id}`, f.id)), h("td", {}, f.blocking ? "genuine" : "note"), h("td", {}, f.closed ? `closed ${f.closedAs || ""}` : "open"))))));
  }
  const turns = view.turns.filter((t) => t.manager === m.path);
  if (turns.length) {
    section(c, "Turns", h("table", {}, h("tbody", {}, turns.slice(-12).reverse().map((t) => h("tr", {},
      h("td", {}, r(`turn:${t.id}`, `${t.pass} ${t.short}`)), h("td", {}, t.role), h("td", {}, t.ended ? duration(Date.parse(t.ended) - Date.parse(t.started)) : "running"))))));
  }
  if (m.declared?.length) {
    section(c, "Dependencies its accepted plan declares for this Pass", h("table", {}, h("tbody", {}, m.declared.map((d) => h("tr", {},
      h("td", {}, d.stage, " of ", r(`manager:${d.owner}`, label(d.owner))), h("td", { class: d.accepted ? "st-done" : "st-waiting" }, d.accepted ? "accepted" : "not accepted"), h("td", { class: "prose" }, d.need))))));
  }
  const kids = m.children?.length ? m.children : null;
  if (kids) section(c, "Children", h("p", {}, kids.map((k, i) => [i ? ", " : "", r(`manager:${k}`, label(k))])));
  if (m.parent) section(c, "Parent", h("p", {}, r(`manager:${m.parent}`, label(m.parent))));
}

function why(c, m) {
  const { r } = c;
  const w = m.wait;
  const box = h("div", { class: "why", role: "group", "aria-label": "Why it waits" });
  if (w.outdated) {
    box.append(h("div", { class: "whyhead" }, "Waiting; the recorded reason no longer holds"));
    box.append(h("p", { style: { margin: "4px 0" } }, `Last recorded reason: waits for ${w.text}. ${w.outdated}`));
    box.append(h("p", { class: "note" }, "The Orchestrator does not record a changed wait while the state stays Waiting, so the record has no newer reason."));
    const open = (m.declared || []).filter((d) => !d.accepted);
    if (open.length) {
      box.append(h("div", {}, "Its accepted plan still needs, unaccepted at this boundary:"),
        h("ul", { class: "chain" }, open.map((d) => h("li", {}, h("span", { class: "arrow" }, "→"), h("span", {}, `the ${d.stage} of `, r(`manager:${d.owner}`, label(d.owner)), ": ", h("span", { class: "prose" }, d.need))))));
    }
    if (w.cause?.length) box.append(h("ul", { class: "chain" }, w.cause.map((st) => h("li", { style: { paddingLeft: `${st.depth * 14}px` } },
      h("span", { class: "arrow" }, "→"), h("span", {}, st.node.startsWith("@") ? h("strong", {}, label(st.node)) : r(`manager:${st.node}`, label(st.node)), " ", st.repeat ? "(already on this path: a cycle of waits)" : st.doing || stateWord(st.state))))));
    return box;
  }
  box.append(h("div", { class: "whyhead" }, `Waits for ${w.text || "its release in this Pass"}`));
  box.append(h("div", { class: "muted" }, `since ${clock(w.sinceAt)}, ${since(w.sinceAt, c.view.boundary.at)} before the boundary`));
  const named = (w.refs || []).map((ref) => r(ref, ref.split(":")[1]));
  if (named.length) box.append(h("div", {}, "Named in the wait: ", named.map((n, i) => [i ? ", " : "", n])));
  if (w.need) box.append(h("div", {}, "Needed: ", h("span", { class: "prose" }, w.need)));
  if (w.agent) {
    box.append(h("div", { class: "note" }, `${label(m.path)} said why (report at record ${w.agent.seq}):`), prose(w.agent.reason));
  }
  if (w.kind === "unrecognized") box.append(h("p", { class: "note" }, "This wait is not in a form the cockpit knows; it names nothing it can follow."));
  if (w.cause?.length) {
    box.append(h("div", { class: "muted", style: { marginTop: "8px" } }, "What has to happen next, as recorded:"));
    box.append(h("ul", { class: "chain" }, w.cause.map((st) => h("li", { style: { paddingLeft: `${st.depth * 14}px` } },
      h("span", { class: "arrow" }, "→"),
      h("span", {}, st.node.startsWith("@") ? h("strong", {}, label(st.node)) : r(`manager:${st.node}`, label(st.node)), " ",
        st.repeat ? "(already on this path: a cycle of waits)" : st.doing || stateWord(st.state))))));
  } else if (w.kind === "children" && !w.on?.length) {
    box.append(h("div", { class: "note" }, "Every child is Done in the record; the Manager's next turn integrates their results."));
  }
  return box;
}

function statusText(p) {
  switch (p.status) {
    case "submitted": return p.checked ? `checked, awaiting ${p.manager === "." ? "the Orchestrator" : "its parent"}` : "submitted, check pending";
    case "accepted": return `accepted by ${label(byNode(p.by))}`;
    case "returned": return `returned by ${label(byNode(p.by))}`;
    default: return p.status;
  }
}

function byNode(by) {
  return !by || by === "the Orchestrator" ? "@orchestrator" : by;
}

// ---------- Proposal ----------

function proposal(c) {
  const { view, id, r } = c;
  const p = view.proposals.find((x) => x.id === id);
  if (!p) {
    title(c, id, "Not submitted up to this boundary.");
    return;
  }
  title(c, `${cap(p.kind)} ${p.id}`, `${label(p.manager)}, ${p.purpose} Pass ${p.pass}`);
  const decider = p.manager === "." ? "the Orchestrator" : label(parentOf(p.manager));
  const steps = h("ol", { class: "steps" },
    h("li", { class: "ok" }, `Submitted at ${clock(p.at)} in `, r(`turn:${p.turn}`, `turn ${shortTurn(p.turn)}`)),
    h("li", { class: p.checked === undefined || p.checked === null ? "now" : p.checked ? "ok" : "bad" },
      p.checked === undefined || p.checked === null ? "Technical check after the turn ends: not recorded yet" : p.checked ? "Passed the Orchestrator's technical check" : "Failed the technical check; it stays its owner's work"),
    h("li", { class: p.status === "accepted" ? "ok" : p.status === "returned" ? "bad" : p.status === "submitted" ? "now" : "" },
      p.status === "submitted" ? `Decision by ${decider}: not recorded yet` : `${cap(p.status)}${p.by ? ` by ${label(byNode(p.by))}` : ""}`));
  c.body.append(steps);
  if (p.reason) section(c, p.status === "accepted" ? "Note" : "Reason", prose(p.reason));
  if (p.stand) c.body.append(h("p", { class: "note" }, "Accepted stand ", h("span", { class: "digest" }, shortDigest(p.stand))));
  const wait = pending(c);
  c.load().then((records) => {
    if (!records) return;
    wait.remove();
    const sub = records.find((x) => x.kind === "proposal.submitted");
    const env = sub?.data?.envelope || {};
    if (env.summary) section(c, "Summary", prose(env.summary));
    if (env.verdict) c.body.append(h("p", {}, "Verdict ", verdictTag(env.verdict)));
    if (env.obligations?.length) {
      section(c, "Obligation coverage", h("table", {}, h("thead", {}, h("tr", {}, h("th", {}, "Definition"), h("th", {}, "Status"), h("th", {}, "Note"))),
        h("tbody", {}, env.obligations.map((o) => h("tr", {}, h("td", {}, o.definition), h("td", {}, o.status), h("td", { class: "prose" }, o.note || ""))))));
    }
    if (env.plan) {
      section(c, "Plan", prose(env.plan.approach));
      if (env.plan.dependencies?.length) {
        section(c, "Declared dependencies", h("ul", {}, env.plan.dependencies.map((d) => h("li", {}, `${d.stage} of `, r(`manager:${d.owner}`, label(d.owner)), ` in ${d.pass}: `, h("span", { class: "prose" }, d.need)))));
      }
      if (env.plan.order) section(c, "Order", prose(env.plan.order));
      if (env.plan.blockers?.length) section(c, "Blockers", h("ul", {}, env.plan.blockers.map((b) => h("li", { class: "prose" }, b))));
    }
    if (env.artifacts?.length) section(c, "Artifacts", h("ul", {}, env.artifacts.map((a) => h("li", { class: "digest" }, a))));
    if (env.evidence?.length) section(c, "Evidence it names", h("ul", {}, env.evidence.map((a) => h("li", { class: "prose" }, a))));
    const raised = sub?.data?.findings || [];
    if (raised.length) section(c, "Findings it raises", h("p", { class: "note" }, "They take effect only when the submission is accepted."), h("ul", {}, raised.map((f) => h("li", { class: "prose" }, `${f.owner}: ${f.observed || f.obligation || JSON.stringify(f)}`))));
    if (p.closes?.length) section(c, "Findings it closes", h("p", {}, p.closes.map((f, i) => [i ? ", " : "", r(`finding:${f}`, f)])));
    if (p.disputes?.length) section(c, "Findings it disputes", h("p", {}, p.disputes.map((f, i) => [i ? ", " : "", r(`finding:${f}`, f)])));
    if (p.children?.length) section(c, "Accepted child results it integrates", h("p", {}, p.children.map((x, i) => [i ? ", " : "", r(`proposal:${x}`, x)])));
    const checked = records.find((x) => x.kind === "proposal.checked" && x.data?.passed === false);
    if (checked?.data?.findings?.length) section(c, "Technical check findings", h("ul", {}, checked.data.findings.map((f) => h("li", {}, typeof f === "string" ? f : JSON.stringify(f)))));
  }).catch((err) => failed(wait, err));
}

// ---------- Finding ----------

function finding(c) {
  const { view, id, r } = c;
  const f = view.findings.find((x) => x.id === id);
  if (!f) {
    title(c, id, "Not recorded up to this boundary.");
    return;
  }
  const raisedIn = view.passes.find((p) => p.id === f.pass);
  title(c, `Finding ${f.id}`, f.blocking ? "Genuine finding: blocks a clean Verify until independently closed" : "Note: recorded, does not block");
  c.body.append(dl([
    ["Owner", r(`manager:${f.owner}`, label(f.owner))],
    ["Raised by", f.raisedBy === "markitect" ? "the Orchestrator (failed required check)" : r(`manager:${f.raisedBy}`, label(f.raisedBy))],
    ["Raised in", r(`pass:${f.pass}`, `${raisedIn?.purpose || ""} Pass ${f.pass}`)],
    f.candidate && ["On candidate", h("span", { class: "digest" }, shortDigest(f.candidate))],
    f.blocking && f.correction && ["Correction", `earliest in ${f.correction}`],
  ]));
  section(c, "Obligation", prose(f.obligation));
  section(c, "Observed", prose(f.observed));
  if (f.disputes?.length) section(c, "Disputes", f.disputes.map((d) => h("div", {}, h("p", { class: "note" }, `${label(d.by)} in ${d.pass}`), prose(d.reason), d.evidence && prose(d.evidence))));
  if (f.closed) {
    const closedIn = view.passes.find((p) => p.id === f.closedPass);
    const same = closedIn?.candidate && f.candidate && closedIn.candidate === f.candidate;
    section(c, `Closed as ${f.closedAs || "closed"}`,
      h("p", {}, "In ", r(`pass:${f.closedPass}`, `${closedIn?.purpose || ""} Pass ${f.closedPass}`),
        closedIn?.candidate ? [" on candidate ", h("span", { class: "digest" }, shortDigest(closedIn.candidate)), same ? " (the same candidate)" : " (a changed candidate)"] : ""),
      prose(f.closedReason), f.closedEvidence && prose(f.closedEvidence));
  } else if (f.blocking) {
    c.body.append(h("p", { class: "why" }, "Open. Only a later accepted review submission of its raising responsibility closes it: resolved on a changed candidate or disproved on the unchanged one."));
  }
}

// ---------- Question ----------

function question(c) {
  const { view, id, r } = c;
  const q = view.questions.find((x) => x.id === id);
  if (!q) {
    title(c, id, "Not asked up to this boundary.");
    return;
  }
  const toUser = q.to === "user";
  title(c, `Question ${q.id}`, `${label(q.from)} to ${toUser ? "the owner (an earlier run recorded this; Runs no longer ask the owner)" : label(q.to)}, ${q.pass}`);
  c.body.append(h("ol", { class: "steps" },
    h("li", { class: "ok" }, `Asked at ${clock(q.asked)}`),
    q.status === "answered"
      ? h("li", { class: "ok" }, `Answered by ${toUser ? "the owner" : label(q.answeredBy || q.to)} at ${clock(q.answeredAt)}`)
      : q.status === "lapsed"
        ? h("li", { class: "bad" }, "Lapsed unanswered when its Pass closed; it cannot be answered any more")
        : h("li", { class: "now" }, "No answer recorded at this boundary")));
  if (q.about) c.body.append(h("p", {}, "Serves ", r(`question:${q.about}`, q.about), ", forwarded along the tree."));
  section(c, "Question", prose(q.text));
  if (q.status === "answered") section(c, "Answer", prose(q.answer));
  const servedBy = view.questions.filter((x) => x.about === q.id);
  if (servedBy.length) section(c, "Forwarded as", h("p", {}, servedBy.map((x, i) => [i ? ", " : "", r(`question:${x.id}`, `${x.id} to ${label(x.to === "user" ? "@user" : x.to)}`)])));
  c.body.append(h("p", { class: "note" }, "The record shows the send and the answer; whether and when the recipient read the question is not recorded."));
}

// ---------- Turn ----------

function turn(c) {
  const { view, id, r } = c;
  const t = view.turns.find((x) => x.id === id);
  if (!t) {
    title(c, shortTurn(id), "Not started up to this boundary.");
    return;
  }
  title(c, `${cap(t.role)} turn ${t.short}`, `${label(t.manager)}, Pass ${t.pass}`);
  const usage = t.usage;
  c.body.append(dl([
    ["Manager", r(`manager:${t.manager}`, label(t.manager))],
    ["Access", t.readOnly ? "read-only" : "writes in its own areas"],
    ["Kind", t.kind],
    t.profile && ["Profile", t.profile],
    ["Started", clock(t.started)],
    ["Ended", t.ended ? `${clock(t.ended)} after ${since(t.started, t.ended)}` : "still running at the boundary"],
    t.ended && ["Reports", t.reports?.length ? t.reports.join(", ") : "none: the turn is incomplete work, not completion"],
    t.ended && ["Container stop", t.stopped ? "observed" : "not observed"],
    t.reason && ["End", t.reason],
    usage && ["Usage", `${usage.model || "model unknown"}, ${usage.outputTokens} output tokens, ${usage.providerTurns} provider steps${usage.costUSD ? `, ${usage.costUSD.toFixed(3)} USD reported` : ""}`],
  ]));
  c.body.append(h("p", {}, h("button", { class: "btn", onclick: () => c.openTranscript(t.id) }, "Open the transcript")));
  const wait = pending(c);
  c.load().then((records) => {
    if (!records) return;
    wait.remove();
    const shown = records.filter((x) => !["turn.start", "turn.end"].includes(x.kind));
    section(c, "Recorded in this turn", shown.length
      ? h("ul", { class: "list" }, shown.map((x) => h("li", { onclick: () => c.select(refOf(x)) }, h("span", { class: "t" }, clock(x.at)), h("span", {}, describe(x)))))
      : h("p", { class: "empty" }, "No report, message or question recorded."));
  }).catch((err) => failed(wait, err));
}

// ---------- Pass ----------

function pass(c) {
  const { view, id, r } = c;
  const p = view.passes.find((x) => x.id === id);
  if (!p) {
    title(c, id, "Not started up to this boundary.");
    return;
  }
  title(c, `${cap(p.purpose || "Pass")} Pass ${p.id}`, p.review ? "Independent Verify with fresh read-only reviewers on one immutable candidate" : p.writes ? "Owned writes" : "Read-only");
  if (p.correction) c.body.append(h("p", {}, "Correction Pass for open genuine findings ", p.addresses.map((f, i) => [i ? ", " : "", r(`finding:${f}`, f)]), "."));
  c.body.append(dl([
    p.started && ["Started", clock(p.started)],
    ["State", p.sealed ? h("span", {}, "sealed ", verdictTag(p.verdict)) : view.status.outcome ? "ended with the run, not sealed" : "running"],
    p.sealedAt && ["Sealed", `${clock(p.sealedAt)} after ${since(p.started, p.sealedAt)}`],
    p.startTree && ["Start stand", h("span", { class: "digest" }, shortDigest(p.startTree))],
    p.candidate && ["Candidate", h("span", { class: "digest" }, shortDigest(p.candidate))],
    p.tree && ["Sealed on", h("span", { class: "digest" }, shortDigest(p.tree))],
  ]));
  if (p.gates?.length) {
    section(c, "Whole-stand gates", h("table", {}, h("tbody", {}, p.gates.map((g) => h("tr", {},
      h("td", {}, g.at), h("td", {}, `Model check ${count(g.modelCheck)}`), h("td", {}, `File check ${count(g.fileCheck)}`))))));
  }
  if (p.checks?.length) {
    section(c, "Required checks on the candidate", h("table", {}, h("tbody", {}, p.checks.map((k) => h("tr", {},
      h("td", {}, r(`check:${p.id}/${k.name}`, k.name)), h("td", { class: k.passed ? "st-done" : "st-fail" }, k.passed ? "passed" : `failed, exit ${k.exitCode}`))))));
  }
  if (p.managers?.length) {
    section(c, "Sealed participation", p.managers.map((m) => h("div", { style: { marginBottom: "10px" } },
      h("div", {}, r(`manager:${m.manager}`, label(m.manager)), " ", verdictTag(m.verdict), " ", m.result ? r(`proposal:${m.result}`, m.result) : ""),
      m.summary && prose(m.summary))));
  } else {
    const parts = view.managers.map((m) => h("li", { class: `st-${m.state || "none"}` }, h("span", { class: "lamp" }), " ", r(`manager:${m.path}`, label(m.path)), ` ${stateWord(m.state)}`));
    if (p.id === view.current) section(c, "Managers now", h("ul", { class: "list" }, parts));
  }
}

function count(n) {
  if (n === undefined || n === null) return "not recorded";
  return n === 0 ? "passed" : `${n} finding${n > 1 ? "s" : ""}`;
}

// ---------- Check ----------

function check(c) {
  const [passId, name] = c.id.split("/");
  title(c, `Check ${name}`, `Required check on the candidate of Pass ${passId}`);
  const wait = pending(c);
  c.load().then((records) => {
    if (!records) return;
    wait.remove();
    const k = records[records.length - 1]?.data;
    if (!k) {
      c.body.append(h("p", { class: "empty" }, "Not recorded up to this boundary."));
      return;
    }
    c.body.append(dl([["Result", k.passed ? "passed" : "failed"], ["Exit code", String(k.exitCode)], k.reason && ["Reason", k.reason]]));
    if (k.output) section(c, "Output", h("pre", {}, k.output));
  }).catch((err) => failed(wait, err));
}

// ---------- Problem ----------

function problem(c) {
  const { view, id, r } = c;
  const p = view.problems.find((x) => x.id === id);
  if (!p) {
    title(c, id, "Not raised up to this boundary.");
    return;
  }
  title(c, `Problem ${p.id}`, `${p.category} problem of ${label(p.manager)}, ${p.pass}`);
  c.body.append(dl([
    ["Raised by", r(`manager:${p.manager}`, label(p.manager))],
    ["Escalated", p.escalated ? "yes, as a question to its parent" : "no"],
    p.question && ["Question", r(`question:${p.question}`, p.question)],
    ["Consequence", p.abort ? "Root ends the run: the model is erroneous or contradictory" : "none recorded"],
  ]));
  section(c, "Text", prose(p.text));
}

// ---------- Single record (message, progress, wait, other) ----------

function record(c) {
  const { r } = c;
  const wait = pending(c);
  c.load().then((records) => {
    if (!records) return;
    wait.remove();
    const x = records[0];
    if (!x) {
      title(c, `Record ${c.id}`, "Not part of the record up to this boundary.");
      return;
    }
    const d = x.data || {};
    switch (x.kind) {
      case "message":
        title(c, "Information", `${label(d.from)} to ${label(d.to)}, ${x.pass}`);
        c.body.append(dl([["From", r(`manager:${d.from}`, label(d.from))], ["To", r(`manager:${d.to}`, label(d.to))], ["Sent", clock(x.at)], x.turn && ["In", r(`turn:${x.turn}`, `turn ${shortTurn(x.turn)}`)]]));
        section(c, "Text", prose(d.text));
        c.body.append(h("p", { class: "note" }, "A recorded send. Delivery into the recipient's next turn and its reading are not recorded."));
        return;
      case "report.progress":
        title(c, "Progress report", `${label(x.manager)}, ${x.pass}`);
        section(c, "Summary", prose(d.summary));
        return;
      case "report.wait":
        title(c, "Named wait", `${label(x.manager)}, ${x.pass}`);
        c.body.append(dl([["Waits for", d.for]]));
        section(c, "Reason", prose(d.reason));
        return;
    }
    title(c, x.kind, `Record ${x.seq} at ${clock(x.at)}`);
    c.body.append(dl([x.pass && ["Pass", r(`pass:${x.pass}`, x.pass)], x.manager && ["Manager", r(`manager:${x.manager}`, label(x.manager))], x.turn && ["Turn", r(`turn:${x.turn}`, shortTurn(x.turn))]]));
    if (x.invalid) c.body.append(h("p", { class: "st-fail" }, `Unreadable: ${x.invalid}`));
    section(c, "Data as recorded", h("pre", {}, JSON.stringify(d, null, 2)));
  }).catch((err) => failed(wait, err));
}

// ---------- Owner, Orchestrator and the run ----------

function owner(c) {
  const { view, r } = c;
  title(c, "Owner", "Recorded questions of earlier runs that Root asked the owner. A Run asks the owner nothing now; this cockpit answers none");
  const qs = view.questions.filter((q) => q.to === "user");
  section(c, "Questions", qs.length
    ? h("table", {}, h("tbody", {}, qs.map((q) => h("tr", {}, h("td", {}, r(`question:${q.id}`, q.id)), h("td", {}, q.status), h("td", { class: "prose" }, clip(q.text, 140))))))
    : h("p", { class: "empty" }, "No question to the owner recorded."));
  c.body.append(h("p", { class: "note" }, "A model Root judges incomplete or contradictory ends the run without publication; the model is amended outside the run."));
}

function orchestrator(c) {
  const { view, r } = c;
  title(c, "Orchestrator", "The deterministic controller: binds inputs, schedules turns, checks and accepts Root's submissions, routes exchanges, runs gates and checks");
  const roots = view.proposals.filter((p) => p.manager === ".");
  section(c, "Decisions on Root's submissions", roots.length
    ? h("table", {}, h("tbody", {}, roots.map((p) => h("tr", {}, h("td", {}, r(`proposal:${p.id}`, p.id)), h("td", {}, `${p.kind}, ${p.pass}`), h("td", {}, p.status)))))
    : h("p", { class: "empty" }, "None recorded yet."));
  section(c, "Passes", h("ul", {}, view.passes.map((p) => h("li", {}, r(`pass:${p.id}`, `${p.id} ${p.purpose || ""}`), p.sealed ? ` sealed ${p.verdict}` : ""))));
}

function overview(c) {
  const { view, r } = c;
  const st = view.status;
  title(c, "Run", view.run.id);
  c.body.append(dl([
    view.run.started && ["Started", `${new Date(view.run.started).toLocaleString()}`],
    view.run.deadline && ["Deadline", clock(view.run.deadline)],
    view.run.agentTurns && ["Budget", `${view.run.agentTurns} Agent turns, ${view.run.passes} Passes`],
    ["Turns so far", String(view.turns.length)],
    view.run.start && ["Start stand", h("span", { class: "digest" }, shortDigest(view.run.start))],
    ["Outcome", st.outcome ? `${st.outcome}${st.reason ? `: ${st.reason}` : ""}` : view.boundary.final ? "not recorded" : "not recorded at this boundary"],
  ]));
  const rep = st.report;
  if (rep) {
    section(c, "Run outcome and publication", dl([
      ["Run outcome", h("strong", { class: rep.outcome === "success" ? "st-done" : rep.outcome === "failed" ? "st-fail" : "st-waiting" }, rep.outcome)],
      rep.publication && ["Receipt", `${rep.publication.outcome}${rep.publication.reason ? `: ${rep.publication.reason}` : ""}`],
      rep.retainedWork && ["Retained work", rep.retainedWork],
      rep.error && ["Error", rep.error],
    ]));
    const lastVerify = [...view.passes].reverse().find((p) => p.review && p.sealed);
    if (rep.publication) {
      section(c, "Read-back per path", h("table", {}, h("tbody", {}, rep.publication.paths.map((p) => h("tr", {},
        h("td", { class: "digest" }, p.path), h("td", { class: p.state === "after" ? "st-done" : "st-fail" }, p.state === "after" ? "published content read back" : p.state === "before" ? "still the captured content" : "other or unreadable"))))));
      if (lastVerify) c.body.append(h("p", { class: "note" }, "Apply publishes only the candidate of its clean Verify: ", r(`pass:${lastVerify.id}`, `Pass ${lastVerify.id}`), lastVerify.candidate ? [", candidate ", h("span", { class: "digest" }, shortDigest(lastVerify.candidate))] : "", ". The receipt records no candidate digest itself."));
    }
  } else if (view.boundary.final && st.outcome) {
    c.body.append(h("p", { class: "note" }, "No report.json: publication and Run outcome are not recorded."));
  }
  const openQ = view.questions.filter((q) => q.status === "open");
  if (openQ.length) section(c, "Open questions", h("ul", {}, openQ.map((q) => h("li", {}, r(`question:${q.id}`, q.id), ` ${label(q.from)} to ${q.to === "user" ? "the owner" : label(q.to)}`))));
  const openF = view.findings.filter((f) => f.blocking && !f.closed);
  if (openF.length) section(c, "Open genuine findings", h("ul", {}, openF.map((f) => h("li", {}, r(`finding:${f.id}`, f.id), ` owner ${label(f.owner)}`))));
  if (view.problems.length) section(c, "Problems", h("ul", {}, view.problems.map((p) => h("li", {}, r(`problem:${p.id}`, p.id), ` ${p.category} of ${label(p.manager)}${p.abort ? ", ends the run" : ""}`))));
  c.body.append(h("div", { class: "gaps" }, h("h3", {}, "What this record does not establish"), h("ul", {}, view.gaps.map((g) => h("li", {}, g.text)))));
}

// ---------- helpers ----------

function describe(x) {
  const d = x.data || {};
  switch (x.kind) {
    case "proposal.submitted": return `Submitted ${d.kind} ${d.proposal}`;
    case "report.progress": return "Reported progress";
    case "report.wait": return `Reported waiting for ${d.for}`;
    case "message": return `Informed ${label(d.to)}`;
    case "question.asked": return `Asked ${d.to === "user" ? "the owner" : label(d.to)} ${d.id}`;
    case "question.answered": return `Answered ${d.id}`;
    case "check.requested": return "Ran an advisory File check";
    case "helper.delegated": return `Delegated to helper ${d.helper}`;
    case "problem.raised": return `Raised ${d.category} problem ${d.id}`;
    case "proposal.withdrawn": return `Withdrew ${d.proposal}`;
  }
  return x.kind;
}

function refOf(x) {
  const d = x.data || {};
  if (x.kind.startsWith("proposal.")) return `proposal:${d.proposal}`;
  if (x.kind.startsWith("question.")) return `question:${d.id}`;
  if (x.kind === "problem.raised") return `problem:${d.id}`;
  return `seq:${x.seq}`;
}

function shortTurn(id) {
  return id ? id.slice(id.lastIndexOf("-") + 1) : "";
}

function parentOf(path) {
  if (path === ".") return "";
  const i = path.lastIndexOf("/");
  return i >= 0 ? path.slice(0, i) : ".";
}

function cap(s) {
  return s ? s[0].toUpperCase() + s.slice(1) : "";
}

function clip(s, n) {
  return s && s.length > n ? s.slice(0, n - 1) + "…" : s || "";
}

// The detail plate: what the record says about the selected Manager, Turn,
// proposal, finding, probe, question, message, Pass or check, up to the
// boundary. Text an Agent wrote is set as prose; everything else is the
// Orchestrator's record. Nothing here infers a delivery, a reading or a success.

import { h, clear, clock, since, label, stateWord, shortDigest, duration } from "./dom.js";
import { activity, managerUsage } from "./run-organization-facts.mjs";
import { passTitle, passRef, plural, usageText, buildText, confirmsModelProblem } from "./runfacts.js";

export function renderDetail(aside, ctx) {
  const token = (aside._token = (aside._token || 0) + 1);
  // A live or replay step draws the plate again. What the owner opened and
  // where they scrolled stays: sections remember their state per reference,
  // and the scroll position is restored while the same reference is shown.
  const top = aside._ref === (ctx.ref || "run") ? aside.scrollTop : 0;
  aside._ref = ctx.ref || "run";
  if (!aside._open) aside._open = new Set();
  const keep = () => {
    if (top && aside.scrollTop < top) aside.scrollTop = top;
  };
  clear(aside);
  const close = h("button", { class: "btn icon close", "aria-label": "Close details", onclick: () => ctx.select(null) }, "×");
  const body = h("div", {});
  aside.append(close, body);
  const ref = ctx.ref || "run";
  const [kind, ...rest] = ref.split(":");
  const id = rest.join(":");
  const r = (target, text) => target ? h("button", { class: "ref", onclick: () => ctx.select(target) }, text) : text;
  const later = (records) => token === aside._token && records;
  const load = async () => {
    const records = later((await ctx.detail(ref)).records);
    // The record arrives after the plate was drawn; the plate grows then.
    if (records) queueMicrotask(() => setTimeout(keep, 0));
    return records;
  };
  const disclose = (key, summary, ...children) => {
    const id = `${ref}|${key}`;
    const el = h("details", {}, h("summary", {}, summary), ...children);
    el.open = aside._open.has(id);
    el.addEventListener("toggle", () => (el.open ? aside._open.add(id) : aside._open.delete(id)));
    return el;
  };
  const c = { ...ctx, r, body, load, id, disclose };
  switch (kind) {
    case "manager": manager(c); break;
    case "proposal": proposal(c); break;
    case "finding": finding(c); break;
    case "question": question(c); break;
    case "turn": turn(c); break;
    case "pass": pass(c); break;
    case "check": check(c); break;
    case "problem": problem(c); break;
    case "probe": probe(c); break;
    case "seq": record(c); break;
    case "owner": owner(c); break;
    case "orchestrator": orchestrator(c); break;
    default: overview(c);
  }
  keep();
}

function title(c, text, kicker) {
  c.body.append(h("h2", {}, text));
  if (kicker) c.body.append(h("p", { class: "kicker" }, kicker));
}

function section(c, name, ...children) {
  c.body.append(h("h3", {}, name), ...children.flat(Infinity).filter((x) => x !== null && x !== undefined && x !== false));
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

// ---------- Observation kinds ----------

// A finding says what it claims. The kinds have distinct labels in quiet
// colors; a record of an earlier build has no kind, so its blocking flag
// decides the label.
const KINDS = {
  deviation: ["Deviation", "obs-deviation"],
  suspected: ["Suspicion", "obs-suspected"],
  model: ["Model problem", "obs-model"],
  suggestion: ["Suggestion", "obs-suggestion"],
};

function kindOf(f) {
  return KINDS[f.kind] || (f.blocking ? ["Genuine finding", "obs-deviation"] : ["Note", "obs-suggestion"]);
}

function obsTag(f) {
  const [text, cls] = kindOf(f);
  return h("span", { class: `obs ${cls}` }, text);
}

const CLASSES = {
  model_error: "Model error: a sourced contradiction or erroneous statement",
  model_gap: "Model gap: a missing material decision or an unusable contract",
  unclassified: "Unclassified: the reviewer gave no class",
};

// effectText states what the finding does to the Run at the boundary.
function effectText(f) {
  if (f.effect === "closed") return `Closed as ${f.closedAs || "closed"}`;
  switch (f.kind) {
    case "suspected": return "Suspicion: a concrete, plausible violation to investigate. It holds the Pass back until it is decided and names no correction yet; a fresh review confirms it as a deviation or model problem or resolves it with evidence.";
    case "model": return "Model problem: it points to the model, not to the work. Once Root confirms it the Run ends without publication; the model is corrected outside the Run.";
    case "suggestion": return "Suggestion: an optional idea. It never blocks, dispatches no correction and authorizes no write in this Run.";
  }
  if (f.subject === "artifact") return "Advisory artifact finding: reported, never repaired by the model's Managers. It does not block.";
  if (f.effect === "advisory") return "Note: recorded, does not block.";
  return "Confirmed deviation: blocks acceptance until it is corrected or independently disproved.";
}

function findingLine(c, f) {
  return h("li", { class: "finding-line" }, c.r(`finding:${f.id}`, f.id), " ", obsTag(f),
    f.class ? h("span", { class: "muted" }, ` ${f.class.replace("_", " ")}`) : null,
    h("span", { class: "muted" }, ` owner ${label(f.owner)}${f.effect === "closed" ? `, closed ${f.closedAs || ""}` : ""}`));
}

// ---------- Probes ----------

const STAND = { candidate: "the candidate", accepted: "the accepted stand", proposal: "a checked proposal", live: "the live stand" };

function outcomeTag(p) {
  if (!p.outcome) return h("span", { class: "tag note" }, "running");
  const cls = p.outcome === "passed" ? "ok" : p.outcome === "failed" ? "fail" : "warn";
  return h("span", { class: `tag ${cls}` }, p.outcome);
}

function dispositionText(p) {
  const d = p.disposition;
  if (d) return `${d.as.replace("_", " ")} by ${label(d.by)}${d.pending ? ", pending an independent review" : ""}`;
  if (p.voidedSeq) return "withdrawn";
  return p.adverse ? "none yet" : "";
}

// probeItem is one probe in a list: its obligation, claim and expectation, the
// stand it ran on, the outcome and what became of an adverse receipt.
function probeItem(c, p, { showPass = false } = {}) {
  const bits = [
    STAND[p.standKind] || p.standKind,
    p.outcome ? `exit ${p.exitCode}` : null,
    p.uncertain ? "uncertain" : null,
    p.timedOut ? "timed out" : null,
    p.rerun ? `repeats ${p.rerun}` : null,
    showPass ? `Pass ${p.pass}` : null,
  ].filter(Boolean);
  const disp = dispositionText(p);
  return h("li", { class: `probe${p.adverse ? " adverse" : ""}` },
    h("div", { class: "probehead" }, c.r(`probe:${p.id}`, `Probe ${p.id}`), " ", outcomeTag(p), h("span", { class: "muted" }, ` ${p.obligation}`)),
    h("div", { class: "prose" }, p.claim),
    p.expected ? [h("div", { class: "muted probelabel" }, "Expected"), h("div", { class: "prose" }, p.expected)] : null,
    h("div", { class: "muted probefoot" }, bits.join(" · "), p.adverse ? ` · disposition: ${disp}` : ""),
    p.adverse && p.disposition?.finding ? h("div", { class: "muted" }, "Finding ", c.r(`finding:${p.disposition.finding}`, p.disposition.finding)) : null);
}

function probeList(c, probes, opts) {
  return h("ul", { class: "probes" }, probes.map((p) => probeItem(c, p, opts)));
}

function probeSummary(probes) {
  const adverse = probes.filter((p) => p.adverse);
  const open = adverse.filter((p) => !p.disposition || p.disposition.pending);
  return `${plural(probes.length, "probe")}: ${probes.filter((p) => p.outcome === "passed").length} passed${adverse.length ? `, ${adverse.length} adverse (${open.length} without an accepted disposition)` : ""}`;
}

function probe(c) {
  const { view, id, r } = c;
  const p = view.probes.find((x) => x.id === id);
  if (!p) {
    title(c, `Probe ${id}`, "Not requested up to this boundary.");
    return;
  }
  title(c, `Probe ${p.id}`, `${p.obligation}, ${STAND[p.standKind] || p.standKind}`);
  c.body.append(h("p", {}, outcomeTag(p), p.adverse ? h("span", { class: "muted" }, ` disposition: ${dispositionText(p)}`) : null));
  const cited = view.findings.filter((f) => f.receipts?.includes(p.id));
  c.body.append(dl([
    ["Requested by", r(`manager:${p.manager}`, label(p.manager))],
    ["In", [r(`pass:${p.pass}`, `Pass ${p.pass}`), p.turn ? [", ", r(`turn:${p.turn}`, `turn ${shortTurn(p.turn)}`)] : ""]],
    p.check && ["Check profile", p.check],
    p.stand && ["Stand", h("span", { class: "digest" }, shortDigest(p.stand))],
    p.path && ["Probe file", h("span", { class: "digest" }, p.path)],
    p.outcome && ["Exit code", `${p.exitCode}${p.timedOut ? ", timed out" : ""}${p.uncertain ? ", uncertain" : ""}`],
    p.rerun && ["Repeats", r(`probe:${p.rerun}`, p.rerun)],
    p.proposal && ["For submission", r(`proposal:${p.proposal}`, p.proposal)],
  ]));
  section(c, "Claim", prose(p.claim));
  section(c, "Expected", prose(p.expected));
  if (p.reason) section(c, "Why the Runtime could not establish an outcome", prose(p.reason));
  const d = p.disposition;
  if (d) {
    section(c, `Disposition: ${d.as.replace("_", " ")}`,
      dl([["By", r(`manager:${d.by}`, label(d.by))], d.proposal && ["Submission", r(`proposal:${d.proposal}`, d.proposal)], d.finding && ["Finding", r(`finding:${d.finding}`, d.finding)],
        d.rerun && ["Passing rerun", r(`probe:${d.rerun}`, d.rerun)], d.passing && ["Passing receipt", r(`probe:${d.passing}`, d.passing)]]),
      d.reason && prose(d.reason), d.evidence && prose(d.evidence),
      d.pending ? h("p", { class: "note" }, "Root disproved its own expectation in this Pass. Nobody has assessed it independently yet, so it keeps the Pass from being clean until a fresh reviewer confirms it or raises a finding.") : null);
  } else if (p.voidedSeq) {
    section(c, "Disposition withdrawn", h("p", {}, `At record ${p.voidedSeq}.`), p.voidedWhy && prose(p.voidedWhy));
  } else if (p.adverse) {
    c.body.append(h("p", { class: "why" }, "An adverse receipt. Only a violation finding that cites it, a reasoned disproof of the probe itself, or a later passing rerun of the same claim on the same stand disposes of it."));
  }
  if (cited.length) section(c, "Findings that cite this receipt", h("ul", {}, cited.map((f) => findingLine(c, f))));
  const used = [];
  for (const pv of view.proposals) {
    for (const o of pv.coverage || []) if (o.evidence?.some((e) => e.receipt === p.id)) used.push([pv, o]);
  }
  if (used.length) section(c, "Cited as evidence", h("ul", {}, used.map(([pv, o]) => h("li", {}, r(`proposal:${pv.id}`, pv.id), ` for ${o.definition}`))));
  const wait = pending(c);
  c.load().then((records) => {
    if (!records) return;
    wait.remove();
    const receipt = records.find((x) => x.kind === "probe.receipt")?.data;
    if (!receipt) {
      c.body.append(h("p", { class: "note" }, "No receipt is recorded at this boundary: the probe was requested and had not ended."));
      const asked = records.find((x) => x.kind === "probe.requested")?.data;
      if (asked?.args?.length) section(c, "Arguments", h("pre", {}, asked.args.join(" ")));
      return;
    }
    if (receipt.args?.length) section(c, "Run as", h("pre", {}, [receipt.executable, ...receipt.args].filter(Boolean).join(" ")));
    if (receipt.content) section(c, `Probe bytes${receipt.contentDigest ? ` (${shortDigest(receipt.contentDigest)})` : ""}`, h("pre", {}, receipt.content));
    if (receipt.output) section(c, `Output${receipt.truncated ? " (cut by the Runtime)" : ""}`, h("pre", {}, receipt.output));
  }).catch((err) => failed(wait, err));
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
  c.body.append(h("p", { class: `st-${m.state || "none"} inline` },
    h("span", { class: "lamp" }), h("strong", {}, stateWord(m.state)), h("span", { class: "muted" }, pv ? `in ${passRef(pv.purpose, pv.id)}` : "")));
  if (m.wait) c.body.append(why(c, m));
  const act = activity(m);
  if (m.turn) {
    section(c, "Active turn", h("p", {}, r(`turn:${m.turn.id}`, `${m.turn.role} turn ${m.turn.short}`), `, ${m.turn.readOnly ? "read-only" : "may write its own areas"}${m.turn.mode ? `, ${m.turn.mode}` : ""}, running for ${since(m.turn.started, view.boundary.at)}`),
      m.turn.role === "reviewer" ? h("p", { class: "note" }, "A reviewer turn assesses the immutable candidate; it does not implement.") : null);
  } else if (act && !m.wait) {
    c.body.append(h("p", { class: "muted" }, act));
  }
  if (m.helpers?.length) section(c, "Helpers", h("ul", {}, m.helpers.map((x) => h("li", {}, `${x.role} ${x.id}`))));
  const binding = view.run.binding?.managers?.find((bound) => bound.manager === m.path);
  if (binding) {
    section(c, "Recorded profiles and team", dl([
      ["Execute", binding.execute?.profile || "not recorded"], ["Independent Verify", binding.verify?.profile || "not recorded"],
      ["Active helpers at most", binding.maxHelpers == null ? "not recorded" : String(binding.maxHelpers)],
    ]), h("ul", {}, (binding.helpers || []).map((helper) => h("li", {}, `${helper.role}: ${helper.profile || "profile not recorded"}, ${helper.writes ? "delegated writes" : "read-only"}`))));
  } else c.body.append(h("p", { class: "note" }, "No profile/team binding recorded for this Manager; today's settings are not used."));
  const usage = managerUsage(view.turns, m.path);
  if (usage.turns) section(c, "Reported usage across Passes", h("p", {}, `${usage.turns} recorded Turns; usage on ${usage.reported}. ${usage.providerTurns > 0 ? `${usage.providerTurns} provider turns reported.` : "Provider-turn count not reported."} ${usage.costUSD > 0 ? `${usage.costUSD.toFixed(2)} USD reported.` : "Cost not reported."}`), h("p", {}, `Models reported: ${usage.models.join(", ") || "none"}`));
  const mine = view.proposals.filter((p) => p.manager === m.path && p.pass === view.current);
  section(c, "Submissions in this Pass", mine.length
    ? h("table", {}, h("tbody", {}, mine.map((p) => h("tr", {}, h("td", {}, r(`proposal:${p.id}`, p.id)), h("td", {}, p.kind), h("td", {}, statusText(p))))))
    : h("p", { class: "empty" }, "None recorded yet."));
  contextSection(c, m);
  const probes = view.probes.filter((p) => p.manager === m.path);
  if (probes.length) section(c, "Probes it ran", h("p", { class: "note" }, probeSummary(probes)), probeList(c, probes, { showPass: true }));
  const own = view.findings.filter((f) => f.owner === m.path);
  if (own.length) section(c, "Findings it owns", h("ul", {}, own.map((f) => findingLine(c, f))));
  const bound = view.bindings.filter((b) => b.manager === m.path);
  if (bound.length) {
    section(c, "Dependencies bound to its work", h("ul", {}, bound.map((b) => h("li", {}, b.cycle
      ? [`Refused: the accepted ${b.stage} of `, r(`manager:${b.on}`, label(b.on)), " would close a cycle"]
      : [`Its ${b.serves || "work"} builds on the accepted ${b.stage} of `, r(`manager:${b.on}`, label(b.on)), ` (${b.proposal})`, b.rebound ? ", rebound after a revision" : ""]))));
  }
  const turns = view.turns.filter((t) => t.manager === m.path);
  if (turns.length) {
    section(c, "Turns", h("table", {}, h("tbody", {}, turns.slice(-12).reverse().map((t) => h("tr", {},
      h("td", {}, r(`turn:${t.id}`, `${t.pass} ${t.short}`)), h("td", {}, t.role),
      h("td", { class: "muted" }, [t.mode, t.context].filter(Boolean).join(" · ")),
      h("td", {}, t.ended ? duration(Date.parse(t.ended) - Date.parse(t.started)) : "running"))))));
  }
  if (m.declared?.length) {
    section(c, "Dependencies its accepted plan declares for this Pass", h("table", {}, h("tbody", {}, m.declared.map((d) => h("tr", {},
      h("td", {}, d.stage, " of ", r(`manager:${d.owner}`, label(d.owner))), h("td", { class: d.accepted ? "st-done" : "st-waiting" }, d.accepted ? "accepted" : "not accepted"), h("td", { class: "prose" }, d.need))))));
  }
  const kids = m.children?.length ? m.children : null;
  if (kids) section(c, "Children", h("p", {}, kids.map((k, i) => [i ? ", " : "", r(`manager:${k}`, label(k))])));
  if (m.parent) section(c, "Parent", h("p", {}, r(`manager:${m.parent}`, label(m.parent))));
}

// contextSection shows what the run bound as the Manager's model context and
// which reference-only Definitions it looked up, Pass by Pass. Binding the
// context says what was selected; it does not show what the Agent understood.
function contextSection(c, m) {
  const { view } = c;
  const ctx = m.context;
  const defs = view.lookups.filter((x) => x.manager === m.path && x.kind === "definition");
  const briefs = view.lookups.filter((x) => x.manager === m.path && x.kind === "brief" && !x.helper);
  if (!ctx && !defs.length && !briefs.length) return;
  const items = [];
  if (ctx) {
    items.push(h("p", {}, [`Brief ${shortDigest(ctx.brief?.replace("sha256:", ""))}`, `${ctx.full.length} in full`, `${ctx.reference.length} reference only`].join(" · ")));
    if (ctx.full.length) items.push(c.disclose("full", `Supplied in full (${ctx.full.length})`, h("ul", { class: "defs" }, ctx.full.map((d) => h("li", {}, d)))));
  } else {
    items.push(h("p", { class: "note" }, "This run recorded no context binding for the Manager."));
  }
  const reference = new Set(ctx?.reference || []);
  const full = new Set(ctx?.full || []);
  const byPass = new Map();
  for (const x of defs) {
    if (!byPass.has(x.pass)) byPass.set(x.pass, new Set());
    byPass.get(x.pass).add(x.id);
  }
  const everLooked = new Set(defs.map((x) => x.id));
  if (reference.size || byPass.size) {
    const rows = view.passes.filter((p) => byPass.has(p.id)).map((p) => {
      const ids = [...byPass.get(p.id)];
      const ref = ids.filter((i) => reference.has(i));
      const again = ids.filter((i) => full.has(i));
      const outside = ids.filter((i) => !reference.has(i) && !full.has(i));
      return h("li", {}, h("strong", {}, `${passTitle(p.purpose)} ${p.id}: `), `${ref.length}${reference.size ? ` of ${reference.size}` : ""} reference-only looked up`,
        again.length ? `, ${again.length} supplied in full read again` : "", outside.length ? `, ${outside.length} outside its context` : "",
        h("div", { class: "muted defnames" }, ids.join(", ")));
    });
    const never = [...reference].filter((i) => !everLooked.has(i));
    items.push(h("p", { class: "note" }, reference.size ? "Reference-only Definitions are listed in the Brief; the Manager reads one by looking it up." : "Lookups the Manager made:"),
      rows.length ? h("ul", { class: "lookups" }, rows) : h("p", { class: "empty" }, "No lookup recorded."),
      never.length ? c.disclose("never", `Reference-only, not looked up (${never.length})`, h("ul", { class: "defs" }, never.map((d) => h("li", {}, d)))) : null);
  }
  if (briefs.length) items.push(h("p", { class: "note" }, `Looked up its Brief again ${plural(briefs.length, "time")}.`));
  section(c, "Model context", ...items);
}

function why(c, m) {
  const { r } = c;
  const w = m.wait;
  const box = h("div", { class: "why", role: "group", "aria-label": "Why it waits" });
  if (w.outdated) {
    box.append(h("div", { class: "whyhead" }, "Waiting; the recorded reason no longer holds"));
    box.append(h("p", { class: "tight" }, `Last recorded reason: waits for ${w.text}. ${w.outdated}`));
    box.append(h("p", { class: "note" }, "The Orchestrator does not record a changed wait while the state stays Waiting, so the record has no newer reason."));
    const open = (m.declared || []).filter((d) => !d.accepted);
    if (open.length) {
      box.append(h("div", {}, "Its accepted plan still needs, unaccepted at this boundary:"),
        h("ul", { class: "chain" }, open.map((d) => h("li", {}, h("span", { class: "arrow" }, "→"), h("span", {}, `the ${d.stage} of `, r(`manager:${d.owner}`, label(d.owner)), ": ", h("span", { class: "prose" }, d.need))))));
    }
    if (w.cause?.length) box.append(chain(c, w.cause));
    return box;
  }
  box.append(h("div", { class: "whyhead" }, `Waits for ${w.text || "its release in this Pass"}`));
  box.append(h("div", { class: "muted" }, `since ${clock(w.sinceAt)}, ${since(w.sinceAt, c.view.boundary.at)} before the boundary`));
  const named = (w.refs || []).map((ref) => r(ref, ref.split(":")[1]));
  if (named.length) box.append(h("div", {}, "Named in the wait: ", named.map((n, i) => [i ? ", " : "", n])));
  if (w.need) box.append(h("div", {}, "Needed: ", h("span", { class: "prose" }, w.need)));
  if (w.agent) {
    box.append(h("div", { class: "note" }, `${label(m.path)} said why (report at record ${w.agent.seq}${w.agent.stage ? `, waiting for the accepted ${w.agent.stage}` : ""}):`), prose(w.agent.reason));
  }
  if (w.kind === "unrecognized") box.append(h("p", { class: "note" }, "This wait is not in a form the cockpit knows; it names nothing it can follow."));
  if (w.cause?.length) {
    box.append(h("div", { class: "muted gap-top" }, "What has to happen next, as recorded:"), chain(c, w.cause));
  } else if (w.kind === "children" && !w.on?.length) {
    box.append(h("div", { class: "note" }, "Every child is Done in the record; the Manager's next turn integrates their results."));
  }
  return box;
}

function chain(c, steps) {
  const { r } = c;
  return h("ul", { class: "chain" }, steps.map((st) => h("li", { style: { paddingLeft: `${st.depth * 14}px` } },
    h("span", { class: "arrow" }, "→"),
    h("span", {}, st.node.startsWith("@") ? h("strong", {}, label(st.node)) : r(`manager:${st.node}`, label(st.node)), " ",
      st.repeat ? "(already on this path: a cycle of waits)" : st.doing || stateWord(st.state)))));
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

const COVERAGE = { done: "no problem found", preserved: "preserved", not_applicable: "not applicable", open: "open" };

// coverageList shows the obligation coverage of a submission: per Definition
// its status and note and the evidence behind it (a probe receipt or an
// inspected source location with the reviewer's reasoning).
function coverageList(c, obligations, { assessment = false } = {}) {
  return h("ul", { class: "coverage" }, obligations.map((o) => h("li", { class: `cov cov-${o.status}` },
    h("div", { class: "covhead" }, h("strong", {}, o.definition), " ", h("span", { class: `tag ${o.status === "open" ? "warn" : "note"}` }, assessment && o.status === "done" ? COVERAGE.done : COVERAGE[o.status] || o.status)),
    o.note ? h("div", { class: "prose" }, o.note) : null,
    (o.evidence || []).map((e) => h("div", { class: "evidence" },
      h("div", {}, e.claim),
      h("div", { class: "muted" }, e.receipt ? ["Probe receipt ", c.r(`probe:${e.receipt}`, e.receipt)] : [e.source ? "Source " : "", e.source ? h("span", { class: "digest" }, e.source) : "", e.location ? ` — ${e.location}` : ""]),
      e.reasoning ? h("div", { class: "prose" }, e.reasoning) : null)))));
}

function proposal(c) {
  const { view, id, r } = c;
  const p = view.proposals.find((x) => x.id === id);
  if (!p) {
    title(c, id, "Not submitted up to this boundary.");
    return;
  }
  title(c, `${cap(p.kind)} ${p.id}`, `${label(p.manager)}, ${passRef(view.passes.find((x) => x.id === p.pass)?.purpose || p.purpose, p.pass)}`);
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
    if (env.obligations?.length) section(c, "Obligation coverage", coverageList(c, env.obligations, { assessment: p.purpose === "assess" }));
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
    if (raised.length) {
      section(c, "Findings it raises", h("p", { class: "note" }, "They take effect only when the submission is accepted."),
        h("ul", {}, raised.map((f) => h("li", {}, obsTag(f), " ", f.receipts?.length ? [f.receipts.map((x, i) => [i ? ", " : "", r(`probe:${x}`, x)]), " "] : "", h("span", { class: "prose" }, `${label(f.owner)}: ${f.observed || f.obligation || JSON.stringify(f)}`)))));
    }
    const disposed = sub?.data?.dispositions;
    if (disposed && Object.keys(disposed).length) {
      section(c, "Dispositions of adverse receipts", h("ul", {}, Object.entries(disposed).map(([receipt, d]) => h("li", {}, r(`probe:${receipt}`, receipt), ` ${String(d.as).replace("_", " ")}: `, h("span", { class: "prose" }, d.reason || d.evidence || "")))));
    }
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
  const [kindText] = kindOf(f);
  title(c, `${kindText} ${f.id}`, effectText(f));
  const confirmsDeviation = f.kind === "deviation" || (!f.kind && f.blocking);
  c.body.append(dl([
    ["Kind", obsTag(f)],
    f.class && ["Class", CLASSES[f.class] || f.class],
    ["Owner", r(`manager:${f.owner}`, label(f.owner))],
    ["Raised by", f.raisedBy === "markitect" ? "the Orchestrator (failed required check)" : r(`manager:${f.raisedBy}`, label(f.raisedBy))],
    ["Raised in", r(`pass:${f.pass}`, passRef(raisedIn?.purpose, f.pass))],
    f.candidate && ["On candidate", h("span", { class: "digest" }, shortDigest(f.candidate))],
    f.subject && ["Subject", f.subject],
    f.definition && ["Definition", f.definition],
    confirmsDeviation && f.correction && ["Correction", `earliest in ${f.correction}`],
    f.receipts?.length && ["Receipts", f.receipts.map((x, i) => [i ? ", " : "", r(`probe:${x}`, x)])],
  ]));
  section(c, "Obligation", prose(f.obligation));
  if (f.expected) section(c, "Expected", prose(f.expected));
  section(c, "Observed", prose(f.observed));
  if (f.disputes?.length) section(c, "Disputes", f.disputes.map((d) => h("div", {}, h("p", { class: "note" }, `${label(d.by)} in ${d.pass}`), prose(d.reason), d.evidence && prose(d.evidence))));
  if (f.closed) {
    const closedIn = view.passes.find((p) => p.id === f.closedPass);
    const same = closedIn?.candidate && f.candidate && closedIn.candidate === f.candidate;
    section(c, `Closed as ${f.closedAs || "closed"}`,
      h("p", {}, "In ", r(`pass:${f.closedPass}`, passRef(closedIn?.purpose, f.closedPass)),
        closedIn?.candidate ? [" on candidate ", h("span", { class: "digest" }, shortDigest(closedIn.candidate)), same ? " (the same candidate)" : " (a changed candidate)"] : ""),
      f.closedAs === "confirmed" ? h("p", { class: "note" }, "A suspicion is closed as confirmed when the same submission raises the deviation or model problem it turned out to be.") : null,
      prose(f.closedReason), f.closedEvidence && prose(f.closedEvidence));
  } else if (f.effect === "blocks") {
    c.body.append(h("p", { class: "why" }, "Open. Only a later accepted review submission of its raising responsibility closes it: resolved on a changed candidate or disproved on the unchanged one."));
  } else if (f.effect === "investigate") {
    c.body.append(h("p", { class: "why" }, "Open and under investigation. It holds the Pass back until a fresh review of the candidate decides it, and it names no correction before it is confirmed."));
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

const CONTEXT = {
  brief: "a fresh conversation: the prompt carries the Brief",
  history: "a continued conversation that received its standing context earlier",
  rehydrated: "a continued conversation that may have lost its context; the Brief was given again",
};

function turn(c) {
  const { view, id, r } = c;
  const t = view.turns.find((x) => x.id === id);
  if (!t) {
    title(c, shortTurn(id), "Not started up to this boundary.");
    return;
  }
  title(c, `${cap(t.role)} turn ${t.short}`, `${label(t.manager)}, Pass ${t.pass}`);
  const usage = t.usage;
  const probes = view.probes.filter((p) => p.turn === t.id);
  const lookups = view.lookups.filter((x) => x.turn === t.id);
  c.body.append(dl([
    ["Manager", r(`manager:${t.manager}`, label(t.manager))],
    ["Access", t.readOnly ? "read-only" : "writes in its own areas"],
    ["Kind", t.kind],
    t.mode && ["Mode", t.mode],
    t.context && ["Context", `${t.context}: ${CONTEXT[t.context] || ""}`],
    t.compacted && ["Compaction", "the provider compacted the conversation during the turn"],
    t.profile && ["Profile", t.profile],
    ["Started", clock(t.started)],
    ["Ended", t.ended ? `${clock(t.ended)} after ${since(t.started, t.ended)}` : "still running at the boundary"],
    t.ended && ["Reports", t.reports?.length ? t.reports.join(", ") : "none: the turn is incomplete work, not completion"],
    t.ended && ["Container stop", t.stopped ? "observed" : "not observed"],
    t.reason && ["End", t.reason],
    usage && ["Usage", `${usage.model || "model unknown"}, ${usage.outputTokens} output tokens, ${usage.providerTurns} provider steps${usage.costUSD ? `, ${usage.costUSD.toFixed(3)} USD reported` : ""}`],
    probes.length && ["Probes", probes.map((p, i) => [i ? ", " : "", r(`probe:${p.id}`, p.id), ` ${p.outcome || "running"}`])],
    lookups.length && ["Lookups", lookups.map((x) => x.kind === "brief" ? "its Brief" : x.id).join(", ")],
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
  const assess = p.purpose === "assess";
  title(c, passRef(p.purpose, p.id), assess ? "Model assessment of the bound model by fresh read-only reviewers, before any work" : p.review ? "Independent Verify with fresh read-only reviewers on one immutable candidate" : p.writes ? "Owned writes" : "Read-only");
  if (assess) assessment(c, p);
  if (p.unchanged) {
    c.body.append(h("p", { class: "round" }, h("strong", {}, "Re-Verify of an unchanged candidate. "),
      "This review judges the very candidate of ", r(`pass:${p.unchanged}`, `Pass ${p.unchanged}`), ": it decides that Pass's disputes and suspicions. It is not a review of a corrected candidate."));
  } else if (p.correction) {
    c.body.append(h("p", { class: "round" }, h("strong", {}, "Correction round. "), "This Pass works on open confirmed findings ", p.addresses.map((f, i) => [i ? ", " : "", r(`finding:${f}`, f)]), ". A review of the corrected candidate follows."));
  }
  c.body.append(dl([
    p.started && ["Started", clock(p.started)],
    ["State", p.sealed ? h("span", {}, "sealed ", verdictTag(p.verdict)) : view.status.outcome ? "ended with the run, not sealed" : "running"],
    p.sealedAt && ["Sealed", `${clock(p.sealedAt)} after ${since(p.started, p.sealedAt)}`],
    p.turns && ["Turns and cost", `${usageText(p)}; ${p.outputTokens || 0} output tokens`],
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
  const probes = view.probes.filter((x) => x.pass === p.id);
  if (probes.length) {
    section(c, "Probes", h("p", { class: "note" }, `${probeSummary(probes)}. A probe is an executed claim of a modeled obligation on ${p.review ? "the immutable candidate" : "a stand of this Pass"}; its receipt is kept whatever a report cites.`), probeList(c, probes));
  } else if (p.review && !assess) {
    section(c, "Probes", h("p", { class: "empty" }, "No probe recorded in this Pass."));
  }
  if (!assess) findingsOfPass(c, p);
  if (p.managers?.length) {
    if (!assess) participation(c, p);
  } else {
    const parts = view.managers.map((m) => h("li", { class: `st-${m.state || "none"}` }, h("span", { class: "lamp" }), " ", r(`manager:${m.path}`, label(m.path)), ` ${stateWord(m.state)}`));
    if (p.id === view.current) section(c, "Managers now", h("ul", { class: "list" }, parts));
  }
}

function participation(c, p) {
  const { r } = c;
  section(c, isAssess(p) ? "Conclusions" : "Sealed participation", p.managers.map((m) => h("div", { class: "participation" },
    h("div", {}, r(`manager:${m.manager}`, label(m.manager)), " ", verdictTag(m.verdict), " ", m.result ? r(`proposal:${m.result}`, m.result) : ""),
    m.summary && prose(m.summary))));
}

function isAssess(p) {
  return p.purpose === "assess";
}

// findingsOfPass lists what the Pass raised and what it closed, by kind.
function findingsOfPass(c, p) {
  const raised = c.view.findings.filter((f) => f.pass === p.id);
  const closed = c.view.findings.filter((f) => f.closedPass === p.id && f.pass !== p.id);
  if (raised.length) section(c, "Observations raised", h("ul", { class: "findings" }, raised.map((f) => findingLine(c, f))));
  if (closed.length) section(c, "Findings closed here", h("ul", { class: "findings" }, closed.map((f) => findingLine(c, f))));
}

// assessment is the Model assessment's verdict: what it found, per obligation
// the Definition, the verdict and the evidence's claim, reasoning and source.
function assessment(c, p) {
  const { view, r } = c;
  const model = view.findings.filter((f) => f.pass === p.id);
  const open = model.filter((f) => f.effect === "blocks");
  const problem = confirmsModelProblem(view, p);
  const box = h("div", { class: `verdict ${problem ? "v-problem" : p.sealed ? "v-ok" : "v-run"}`, role: "group", "aria-label": "Assessment verdict" });
  if (problem) {
    box.append(h("div", { class: "vhead" }, "Model problem confirmed — nothing published."), h("div", {}, "Correct the model and start a new run."));
    if (view.status.reason) box.append(prose(view.status.reason));
  } else if (p.sealed && p.verdict === "satisfied") {
    box.append(h("div", { class: "vhead" }, "No material model problem found"), h("div", { class: "muted" }, "The recorded coverage found nothing unresolved. That is not proof of exhaustive completeness, and it does not replace the artifact Verify."));
  } else if (p.sealed) {
    box.append(h("div", { class: "vhead" }, "Sealed unclean"), h("div", { class: "muted" }, "The assessment did not seal clean, for instance because a dispute or an open suspicion is undecided. No model problem is confirmed by this record."));
  } else {
    box.append(h("div", { class: "vhead" }, "Assessment in progress"), h("div", { class: "muted" }, open.length ? `${plural(open.length, "model problem")} raised so far.` : "No model problem raised so far."));
  }
  c.body.append(box);
  if (p.managers?.length) participation(c, p);
  if (model.length) {
    section(c, "Model observations", h("ul", { class: "findings" }, model.map((f) => h("li", { class: "finding-line" }, c.r(`finding:${f.id}`, f.id), " ", obsTag(f),
      f.class ? h("span", { class: `class class-${f.class}` }, ` ${f.class.replace("_", " ")}`) : null,
      h("div", { class: "muted" }, `Definition ${f.obligation} · owner ${label(f.owner)}${f.effect === "closed" ? ` · closed ${f.closedAs || ""}` : ""}`),
      h("div", { class: "prose" }, f.observed)))));
  }
  // Each Manager's latest submission with coverage; its result over its basis.
  const bySubmitter = new Map();
  for (const s of view.proposals.filter((x) => x.pass === p.id && x.coverage?.length)) {
    const old = bySubmitter.get(s.manager);
    if (!old || s.kind === "result" || old.kind !== "result") bySubmitter.set(s.manager, s);
  }
  for (const [m, s] of bySubmitter) {
    section(c, `Assessment by ${label(m)}`, h("p", { class: "note" }, "Submission ", r(`proposal:${s.id}`, s.id), `, ${plural(s.coverage.length, "obligation")}`), coverageList(c, s.coverage, { assessment: true }));
  }
  const bare = [...new Set(view.proposals.filter((x) => x.pass === p.id && x.kind === "result" && !x.coverage?.length).map((x) => x.manager))];
  if (bare.length) c.body.append(h("p", { class: "note" }, `${bare.map(label).join(", ")} submitted a result with no own Definitions to cover.`));
  if (!bySubmitter.size) c.body.append(h("p", { class: "empty" }, "No submission with obligation coverage is recorded at this boundary."));
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

// ---------- Single record (message, progress, wait, lookup, other) ----------

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
        c.body.append(dl([["Waits for", d.stage ? `the accepted ${d.stage} of ${label(d.for)}` : d.for]]));
        section(c, "Reason", prose(d.reason));
        return;
      case "lookup.definition":
      case "lookup.brief":
        title(c, x.kind === "lookup.brief" ? "Brief lookup" : "Definition lookup", `${label(x.manager)}, ${x.pass}`);
        c.body.append(dl([
          d.id && ["Definition", d.id], d.owner && ["Owned by", r(`manager:${d.owner}`, label(d.owner))],
          ["Looked up by", r(`manager:${x.manager}`, label(x.manager))], x.turn && ["In", r(`turn:${x.turn}`, `turn ${shortTurn(x.turn)}`)],
          ["Answer", `${d.bytes} bytes${d.refused ? ", refused as too large" : ""}${d.helper ? `, for helper ${d.helper}` : ""}`],
          d.answer && ["Answer digest", h("span", { class: "digest" }, shortDigest(String(d.answer).replace("sha256:", "")))],
        ]));
        c.body.append(h("p", { class: "note" }, "The record keeps the answer's digest and size, not its text. The text the Agent read is in the turn's transcript."));
        return;
      case "dependency.bound":
      case "dependency.unbound":
        title(c, x.kind === "dependency.bound" ? "Dependency bound" : "Dependency refused", `${label(x.manager)}, ${x.pass}`);
        c.body.append(dl([["Work of", r(`manager:${x.manager}`, label(x.manager))], ["Builds on", [`the accepted ${d.stage} of `, r(`manager:${d.on}`, label(d.on))]], d.proposal && ["Submission", r(`proposal:${d.proposal}`, d.proposal)],
          d.stand && ["Stand", h("span", { class: "digest" }, shortDigest(d.stand))], d.cycle && ["Cycle", d.cycle]]));
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
  if (!view.ownerChannel) {
    title(c, "Owner", "Static project context outside the closed Run");
    c.body.append(h("p", { class: "note" }, "The owner maintains the model and starts Runs outside execution. No question, answer or live external channel crosses Root during this Run."));
    return;
  }
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
  const checks = view.run.binding?.checks;
  if (checks) section(c, "Checks bound at run.start", checks.length ? h("ul", {}, checks.map((check) => h("li", {}, h("strong", {}, check.name), `: ${check.command}, ${check.timeout}${check.requiredBy ? `; required by ${check.requiredBy}` : ""}`))) : h("p", { class: "note" }, "No checks in the recorded binding."));
  const roots = view.proposals.filter((p) => p.manager === ".");
  section(c, "Decisions on Root's submissions", roots.length
    ? h("table", {}, h("tbody", {}, roots.map((p) => h("tr", {}, h("td", {}, r(`proposal:${p.id}`, p.id)), h("td", {}, `${p.kind}, ${p.pass}`), h("td", {}, p.status)))))
    : h("p", { class: "empty" }, "None recorded yet."));
  section(c, "Passes", h("ul", {}, view.passes.map((p) => h("li", {}, r(`pass:${p.id}`, `${p.id} ${passTitle(p.purpose)}`), p.sealed ? ` sealed ${p.verdict.replace("_", " ")}` : ""))));
}

function overview(c) {
  const { view, r } = c;
  const st = view.status;
  const run = view.run;
  title(c, "Run", view.run.id);
  c.body.append(dl([
    run.started && ["Started", `${new Date(run.started).toLocaleString()}`],
    run.strategy && ["Strategy", `${run.strategy}${run.work?.length ? `: ${run.work.join(" → ")} → verify` : ""}`],
    run.build && ["Build", buildText(run.build)],
    run.source && ["Source", h("span", { class: "digest" }, run.source.slice(0, 12))],
    run.agentTurns && ["Budget", `${run.agentTurns} Agent turns, ${run.passes} Passes${run.executionTime ? `, ${run.executionTime} execution time` : ""}`],
    run.deadline && ["Deadline", clock(run.deadline)],
    ["Turns so far", String(view.turns.length)],
    run.start && ["Start stand", h("span", { class: "digest" }, shortDigest(run.start))],
    ["Outcome", st.outcome ? `${st.outcome}${st.reason ? `: ${st.reason}` : ""}` : view.boundary.final ? "not recorded" : "not recorded at this boundary"],
  ]));
  if (st.ending === "model_problem") {
    c.body.append(h("p", { class: "why" }, h("strong", {}, "Model problem confirmed — nothing published. "), "Correct the model and start a new run."));
  }
  const rep = st.report;
  if (rep) {
    const pub = rep.publication;
    section(c, "Result and publication", dl([
      ["Run", rep.run?.status ? h("strong", { class: rep.outcome === "success" ? "st-done" : rep.outcome === "failed" ? "st-fail" : "st-waiting" }, rep.run.status) : rep.outcome],
      ["Publication", pub ? `${pub.outcome === "no_change" ? "no change" : pub.outcome}${pub.reason ? `: ${pub.reason}` : ""}` : "nothing published"],
      rep.retainedWork && ["Retained work", rep.retainedWork],
      rep.error && ["Error", rep.error],
    ]));
    if (pub?.outcome === "no_change") {
      c.body.append(h("p", { class: "note" }, "The verified candidate equals the captured source. Nothing was written, and the guard held."));
    } else if (pub?.paths?.length) {
      const lastVerify = [...view.passes].reverse().find((p) => p.review && p.purpose !== "assess" && p.sealed);
      section(c, "Read-back per path", h("table", {}, h("tbody", {}, pub.paths.map((p) => h("tr", {},
        h("td", { class: "digest" }, p.path), h("td", { class: p.state === "after" ? "st-done" : "st-fail" }, p.state === "after" ? "published content read back" : p.state === "before" ? "still the captured content" : "other or unreadable"))))));
      if (lastVerify) c.body.append(h("p", { class: "note" }, "Apply publishes only the candidate of its clean Verify: ", r(`pass:${lastVerify.id}`, `Pass ${lastVerify.id}`), lastVerify.candidate ? [", candidate ", h("span", { class: "digest" }, shortDigest(lastVerify.candidate))] : "", ". The receipt records no candidate digest itself."));
    }
    const rr = rep.run;
    if (rr) {
      const rows = [
        ["Probes", rr.probes?.total ? `${rr.probes.total} receipts: ${rr.probes.passed} passed${rr.probes.failed ? `, ${rr.probes.failed} failed` : ""}${rr.probes.error ? `, ${rr.probes.error} error` : ""}${rr.probes.undisposed ? `, ${rr.probes.undisposed} adverse without an accepted disposition` : ""}` : "none"],
        ["Turns", String(rr.turns)],
        rr.usage?.providerTurns && ["Usage", `${rr.usage.model || "model unknown"}, ${rr.usage.providerTurns} provider steps${rr.usage.costUSD ? `, ${rr.usage.costUSD.toFixed(2)} USD reported` : ""}`],
      ];
      section(c, "In the report", dl(rows));
      if (rr.problems?.length) section(c, "Problems in the report", h("ul", {}, rr.problems.map((p) => h("li", {}, `${p.category} problem of ${label(p.manager)}${p.abort ? ", ended the run" : ""}: `, h("span", { class: "prose" }, clip(p.text, 240))))));
    }
  } else if (view.boundary.final && st.outcome) {
    c.body.append(h("p", { class: "note" }, "No report.json: the publication is not recorded."));
  }
  if (view.probes.length) section(c, "Probes", h("p", { class: "note" }, probeSummary(view.probes)));
  const openQ = view.questions.filter((q) => q.status === "open");
  if (openQ.length) section(c, "Open questions", h("ul", {}, openQ.map((q) => h("li", {}, r(`question:${q.id}`, q.id), ` ${label(q.from)} to ${q.to === "user" ? "the owner" : label(q.to)}`))));
  const blocking = view.findings.filter((f) => f.effect === "blocks");
  if (blocking.length) section(c, "Open findings that block", h("ul", { class: "findings" }, blocking.map((f) => findingLine(c, f))));
  const suspicions = view.findings.filter((f) => f.effect === "investigate");
  if (suspicions.length) section(c, "Suspicions to investigate", h("ul", { class: "findings" }, suspicions.map((f) => findingLine(c, f))));
  if (view.problems.length) section(c, "Problems", h("ul", {}, view.problems.map((p) => h("li", {}, r(`problem:${p.id}`, p.id), ` ${p.category} of ${label(p.manager)}${p.abort ? ", ends the run" : ""}`))));
  // What the record does not establish is a section to open, not a wall of text.
  const gaps = c.disclose("gaps", `What this record does not establish (${view.gaps.length})`, h("ul", {}, view.gaps.map((g) => h("li", {}, g.text))));
  gaps.classList.add("gaps");
  c.body.append(gaps);
}

// ---------- helpers ----------

function describe(x) {
  const d = x.data || {};
  switch (x.kind) {
    case "proposal.submitted": return `Submitted ${d.kind} ${d.proposal}`;
    case "report.progress": return "Reported progress";
    case "report.wait": return `Reported waiting for ${d.stage ? `the accepted ${d.stage} of ${label(d.for)}` : d.for}`;
    case "message": return `Informed ${label(d.to)}`;
    case "question.asked": return `Asked ${d.to === "user" ? "the owner" : label(d.to)} ${d.id}`;
    case "question.answered": return `Answered ${d.id}`;
    case "check.requested": return "Ran an advisory File check";
    case "helper.delegated": return `Delegated to helper ${d.helper}`;
    case "problem.raised": return `Raised ${d.category} problem ${d.id}`;
    case "proposal.withdrawn": return `Withdrew ${d.proposal}`;
    case "lookup.definition": return `Looked up ${d.id}`;
    case "lookup.brief": return "Looked up its Brief again";
    case "probe.requested": return `Requested probe ${d.id} for ${d.obligation}`;
    case "probe.receipt": return `Probe ${d.id}: ${d.outcome}`;
    case "probe.disposed": return d.voided ? `Probe ${d.receipt} disposition withdrawn` : `Probe ${d.receipt} disposed as ${String(d.disposition?.as).replace("_", " ")}`;
    case "dependency.bound": return `Bound its work to the accepted ${d.stage} of ${label(d.on)}`;
  }
  return x.kind;
}

function refOf(x) {
  const d = x.data || {};
  if (x.kind.startsWith("proposal.")) return `proposal:${d.proposal}`;
  if (x.kind.startsWith("question.")) return `question:${d.id}`;
  if (x.kind === "problem.raised") return `problem:${d.id}`;
  if (x.kind === "probe.requested" || x.kind === "probe.receipt") return `probe:${d.id}`;
  if (x.kind === "probe.disposed") return `probe:${d.receipt}`;
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

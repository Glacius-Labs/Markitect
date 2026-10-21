// Wording the run inspector and the run list share: how a Run's outcome, its
// publication, its strategy and the cost of a Pass are said. The words follow
// the current Apply concept: a Run is closed (nobody answers from outside it),
// starts with a Model assessment, and a confirmed model problem ends it
// without publication.

const OUTCOME_CLASS = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" };

export function passTitle(purpose) {
  if (purpose === "assess") return "Model assessment";
  return purpose ? purpose[0].toUpperCase() + purpose.slice(1) : "Pass";
}

export function plural(n, word) {
  return `${n} ${word}${n === 1 ? "" : "s"}`;
}

// publicationText says what a publication receipt establishes. A no-change
// publication wrote nothing, so it has no read-back to count.
export function publicationText(pub) {
  if (!pub) return "nothing published";
  if (pub.outcome === "no_change") return "no change — nothing to write";
  const after = pub.paths.filter((p) => p.state === "after").length;
  if (pub.outcome === "applied") return `published, ${after}/${pub.paths.length} read back`;
  return `${pub.outcome}, ${after}/${pub.paths.length} read back`;
}

// outcomeFacts is the outcome pill of a view: its words, its class and the
// longer statement for the tooltip.
export function outcomeFacts(v) {
  const st = v.status;
  const rep = st.report;
  if (st.ending === "model_problem") {
    return { text: "Model problem confirmed", cls: "st-waiting", title: "A Model assessment or Root confirmed a material model problem. The Run ended without publication; correct the model and start a new run." };
  }
  if (rep) {
    const status = rep.run?.status || st.outcome || rep.outcome;
    const pub = rep.publication;
    const word = { verified: "Verified", incomplete: "Incomplete", failed: "Failed" }[status] || status;
    const tail = rep.run?.status === "verified" ? ` · ${pub ? (pub.outcome === "no_change" ? "no change" : pub.outcome) : "not published"}` : "";
    return { text: `${word}${tail}`, cls: OUTCOME_CLASS[rep.outcome] || OUTCOME_CLASS[status] || "", title: `The Run ended ${status}; ${publicationText(pub)}.` };
  }
  if (st.outcome) return { text: `Outcome recorded: ${st.outcome}`, cls: OUTCOME_CLASS[st.outcome] || "", title: "The Orchestrator recorded this outcome; report.json is not written yet or absent." };
  return { text: v.boundary.final ? "No outcome recorded" : "In progress at this boundary", cls: v.boundary.final ? "" : "st-working", title: "What the record shows at this boundary." };
}

// outcomeOfSummary is the same pill for one row of the run list.
export function outcomeOfSummary(r, run) {
  if (r.ending === "model_problem") return { text: "Model problem confirmed", cls: "st-waiting" };
  const result = r.runOutcome || r.outcome;
  if (result) {
    const status = r.runStatus || r.outcome;
    const word = { verified: "Verified", incomplete: "Incomplete", failed: "Failed" }[status] || status;
    const tail = r.runOutcome ? (status === "verified" ? ` · ${r.publication ? (r.publication === "no_change" ? "no change" : r.publication) : "not published"}` : "") : ", no report";
    return { text: `${word}${tail}`, cls: OUTCOME_CLASS[r.runOutcome || r.outcome] || "" };
  }
  if (run === "ended-without-report") return { text: "ended without outcome or report", cls: "st-fail" };
  if (run === "running") return { text: "running", cls: "st-working" };
  return { text: "no outcome recorded", cls: "st-none" };
}

// strategyText names the composition a Run was bound to.
export function strategyText(run) {
  if (!run.strategy) return "";
  return run.strategy;
}

// buildText shortens a build identity such as 0.14.1+b9cdd6d893fa… to the
// version and the first eight characters of the commit.
export function buildText(build) {
  if (!build) return "";
  const [version, commit] = build.split("+");
  return commit ? `${version}+${commit.slice(0, 8)}` : version;
}

// usageText is the turns and cost of a Pass, summed from its turns.
export function usageText(p) {
  if (!p.turns) return "";
  const cost = p.costUSD ? `, ${p.costUSD.toFixed(2)} USD reported` : "";
  return `${plural(p.turns, "turn")}${cost}`;
}

export function digest(d) {
  return d ? d.slice(0, 12) : "";
}

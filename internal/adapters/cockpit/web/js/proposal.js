// One proposal: what it changes, what the structural check found, and the
// owner's decision. Accepting is a deliberate two-step action; the cockpit
// shows the publication's own receipt and never calls a proposal accepted
// that the publication did not apply. The view mounts into any container: the
// proposals inbox and the model session both show it.

import { api } from "./api.js";
import { h, clear, append, when, announce, ticks } from "./dom.js";
import { refusal } from "./settings.js";
import { refreshState } from "./shell.js";

export const stateWords = { proposed: "Waiting for your decision", publishing: "Being written", accepted: "Accepted and written", discarded: "Discarded", failed: "Not accepted" };
export const stateClass = { proposed: "st-waiting", publishing: "st-working", accepted: "st-done", discarded: "st-none", failed: "st-fail" };

// mountProposal reads a proposal into container and keeps it current while it
// is being written. It returns a function that stops reading.
export function mountProposal(container, id, { compact = false } = {}) {
  let poll = null;
  let stopped = false;
  const draw = (p, focus) => {
    if (stopped || !container.isConnected) return;
    clearTimeout(poll);
    // A publication in progress is read again until it has its receipt; a
    // failed read keeps the last state and tries again.
    if (p.state === "publishing") poll = setTimeout(() => api.proposal(p.id).then((q) => draw(q, q.state === "publishing" ? null : ".outcome.big"), () => draw(p)), 2000);
    append(clear(container), proposalView(p, draw, compact));
    if (focus) container.querySelector(focus)?.focus();
  };
  container.classList.add("stack");
  clear(container).append(h("p", { class: "note", role: "status" }, "Reading the proposal…"));
  api.proposal(id).then((p) => draw(p), (err) => {
    if (stopped || !container.isConnected) return;
    clear(container).append(h("p", { role: "alert", class: "refusal" }, /no proposal/.test(err.message)
      ? `This cockpit holds no proposal ${id}. Proposals live in the running cockpit and end with it; what was accepted is in the working tree.`
      : refusal(err)));
  });
  return () => {
    stopped = true;
    clearTimeout(poll);
  };
}

function proposalView(p, draw, compact) {
  const passed = p.check.status === "passed";
  return [
    h("div", { class: "block" },
      h("p", { class: `outcome big ${stateClass[p.state] || ""}`, tabindex: -1, style: { margin: 0 } }, h("span", { class: "lamp" }), stateWords[p.state] || p.state),
      h("dl", { class: "facts" },
        h("dt", {}, "Proposal"), h("dd", {}, h("span", { class: "coord" }, p.id), " ", p.title),
        h("dt", {}, "Made"), h("dd", {}, `${when(p.created)} from the committed source at `, h("span", { class: "digest" }, p.head.slice(0, 12))),
        p.decided && [h("dt", {}, "Decided"), h("dd", {}, when(p.decided))],
        h("dt", {}, "Changes"), h("dd", {}, p.changes.map((c) => `${c.path} ${c.action}`).join(", ")))),
    receipt(p),
    p.state === "proposed" ? decision(p, passed, draw) : after(p, compact),
    p.summary && h("div", { class: "block" }, h("h3", {}, "The Agent's summary"),
      h("p", { class: "note", style: { margin: 0 } }, "Written by the Agent of ", h("a", { href: "#/session" }, `session ${p.session}`), "; the difference and the check below are what the cockpit observed."),
      h("div", { class: "prose quote agent" }, p.summary)),
    checkBlock(p.check),
    h("div", { class: "block" }, h("h3", {}, "Difference"), ...p.changes.map(fileDiff)),
  ];
}

function checkBlock(check) {
  const passed = check.status === "passed";
  const block = h("div", { class: "block" },
    h("h3", {}, "Structural check"),
    h("p", { class: `outcome ${passed ? "st-done" : "st-fail"}`, style: { margin: 0 } }, h("span", { class: "lamp" }),
      passed ? "passed: Model check, registry, File check, execution settings and context selection accept the proposed tree" : `${check.status.replace("_", " ")}: the proposal cannot be accepted`));
  if (check.findings?.length) {
    block.append(h("ul", { class: "findings" }, check.findings.map((f) => h("li", {},
      h("span", { class: "fstage" }, f.stage), " ", h("span", { class: "digest" }, f.code), f.path && [" at ", h("span", { class: "digest" }, f.path)],
      h("div", {}, f.message)))));
  }
  return block;
}

// fileDiff shows one changed file as numbered old and new lines; removed
// lines come before the lines that replace them, and long unchanged runs are
// cut to a gap.
function fileDiff(d) {
  const box = h("figure", { class: "diff" }, h("figcaption", {}, h("span", { class: "digest" }, d.path), ` ${d.action}`));
  if (d.binary) {
    box.append(h("p", { class: "muted" }, "Not text; the difference is not shown."));
    return box;
  }
  const rows = (d.lines || []).map((l) => {
    if (l.gap) return h("tr", { class: "gap" }, h("td", { colspan: 4 }, `${l.gap} unchanged line${l.gap === 1 ? "" : "s"}`));
    const cls = l.op === "+" ? "add" : l.op === "-" ? "del" : "ctx";
    const word = l.op === "+" ? "added" : l.op === "-" ? "removed" : "";
    return h("tr", { class: cls },
      h("td", { class: "ln" }, l.old || ""), h("td", { class: "ln" }, l.new || ""),
      h("td", { class: "op" }, l.op === "=" ? "" : l.op, word && h("span", { class: "visually-hidden" }, word)),
      h("td", { class: "code" }, l.text));
  });
  box.append(h("div", { class: "diffscroll", tabindex: 0, role: "region", "aria-label": `Difference of ${d.path}` },
    h("table", {}, h("thead", { class: "visually-hidden" }, h("tr", {}, h("th", {}, "Old line"), h("th", {}, "New line"), h("th", {}, "Change"), h("th", {}, "Text"))), h("tbody", {}, rows))));
  return box;
}

function decision(p, passed, draw) {
  const section = h("section", { class: "decide ticked", "aria-label": "Your decision" }, ticks());
  const status = h("p", { class: "note", role: "status", tabindex: -1, style: { margin: 0 } });
  const actions = h("div", { class: "actions" });
  const paths = p.changes.map((c) => c.path).join(", ");
  const decide = async (what, buttons) => {
    buttons.forEach((b) => { b.disabled = true; });
    status.textContent = what === "accept"
      ? "Locking the state directories, checking that the source is unchanged, writing and reading back. The publication compares the working tree before and after, which takes a while on a large repository. Leaving the page does not interrupt it."
      : "Discarding…";
    try {
      const result = await api.decide(p.id, what);
      announce(`Proposal ${p.id}: ${stateWords[result.state] || result.state}.`);
      draw(result, ".outcome.big");
      // An accepted change is in the working tree: observe the project again.
      if (result.state === "accepted") api.observeProject().catch(() => {});
      refreshState();
    } catch (err) {
      buttons.forEach((b) => { b.disabled = false; });
      status.textContent = `${what === "accept" ? "Not written" : "Not discarded"}: ${refusal(err)}`;
      status.focus();
    }
  };
  const choose = () => {
    const accept = h("button", { class: "btn act", type: "button", disabled: !passed }, "Accept");
    const discard = h("button", { class: "btn", type: "button" }, "Discard");
    accept.addEventListener("click", confirm);
    discard.addEventListener("click", () => decide("discard", [accept, discard]));
    clear(actions).append(accept, discard);
    status.textContent = passed ? "Accepting writes the change into the working tree; nothing is committed and no Apply starts." : "Accepting needs a passed structural check. Discard this proposal or make a new one.";
  };
  const confirm = () => {
    const write = h("button", { class: "btn act", type: "button" }, `Write ${paths}`);
    const back = h("button", { class: "btn", type: "button" }, "Back");
    write.addEventListener("click", () => decide("accept", [write, back]));
    back.addEventListener("click", () => { choose(); actions.firstChild.focus(); });
    clear(actions).append(write, back);
    status.textContent = `This writes ${paths} into the working tree while no observed Apply runs. It refuses when the committed source changed since the proposal. Nothing is committed and no Apply starts.`;
    write.focus();
  };
  choose();
  section.append(h("h3", {}, "Your decision"), actions, status);
  return section;
}

// receipt is the publication's own outcome with each path's read-back state.
function receipt(p) {
  if (!p.receipt && !p.error) return null;
  const box = h("div", { class: "block" }, h("h3", {}, "Publication"));
  if (p.error) box.append(h("p", { class: "refusal", role: "alert" }, p.error));
  if (p.receipt) {
    box.append(h("dl", { class: "facts" },
      h("dt", {}, "Outcome"), h("dd", {}, p.receipt.outcome, p.receipt.reason && h("span", { class: "muted" }, ` (${p.receipt.reason})`)),
      p.receipt.paths.map((x) => [h("dt", {}, h("span", { class: "digest" }, x.path)), h("dd", {}, readBack(x.state))])));
  }
  return box;
}

function readBack(state) {
  return {
    after: "read back as proposed",
    before: "read back unchanged",
    other: "holds other content or was not readable; inspect it",
  }[state] || state;
}

function after(p) {
  if (p.state === "accepted") {
    return h("p", { class: "note", style: { margin: 0 } }, "The change is in the working tree, not committed. Commit it to make it the source of the next Apply and of the next proposal; until then the cockpit makes no new proposal.");
  }
  if (p.state === "failed") {
    return h("p", { class: "note", style: { margin: 0 } }, p.receipt && ["partial", "unknown"].includes(p.receipt.outcome)
      ? "The working tree may hold part of the change. Inspect the paths above before committing or proposing again."
      : "Nothing was written. Make the proposal again from the current committed source.");
  }
  return null;
}

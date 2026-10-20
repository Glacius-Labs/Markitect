// One proposal: what it changes, what the structural check found, and the
// owner's decision. Accepting is a deliberate two-step action; the cockpit
// shows the publication's own receipt and never calls a proposal accepted
// that the publication did not apply.

import { api } from "./api.js";
import { h, clear, append, when, announce } from "./dom.js";
import { top, refusal } from "./settings.js";

const stateWords = { proposed: "Waiting for your decision", publishing: "Being written", accepted: "Accepted and written", discarded: "Discarded", failed: "Not accepted" };
const stateClass = { proposed: "st-waiting", publishing: "st-working", accepted: "st-done", discarded: "st-none", failed: "st-fail" };

export function renderProposal(app, id) {
  document.title = `Proposal ${id} – Markitect cockpit`;
  clear(app);
  const main = h("main", { id: "main", class: "runs authoring" });
  app.append(main);
  main.append(top(`Proposal ${id}`), h("p", { class: "lead", role: "status" }, "Reading the proposal…"));
  let poll = null;
  const draw = (p, focus) => {
    if (!main.isConnected) return;
    // A publication in progress is read again until it has its receipt.
    clearTimeout(poll);
    if (p.state === "publishing") poll = setTimeout(() => api.proposal(p.id).then((q) => draw(q, q.state === "publishing" ? null : ".outcome.big"), () => {}), 2000);
    append(clear(main), [top(`Proposal ${p.id}: ${p.title}`), proposalBody(p, draw)]);
    if (focus) main.querySelector(focus)?.focus();
  };
  api.proposal(id).then((p) => draw(p), (err) => {
    if (!main.isConnected) return;
    clear(main).append(top(`Proposal ${id}`), h("p", { role: "alert", class: "refusal" }, /no proposal/.test(err.message)
      ? `This cockpit holds no proposal ${id}. Proposals live in the running cockpit and end with it; what was accepted is in the working tree.`
      : refusal(err)));
  });
}

function proposalBody(p, draw) {
  const passed = p.check.status === "passed";
  return [
    h("p", { class: `outcome big ${stateClass[p.state] || ""}`, tabindex: -1 }, h("span", { class: "lamp" }), stateWords[p.state] || p.state),
    h("dl", { class: "facts" },
      h("dt", {}, "Made"), h("dd", {}, `${when(p.created)} from the committed source at `, h("span", { class: "digest" }, p.head.slice(0, 12))),
      p.decided && [h("dt", {}, "Decided"), h("dd", {}, when(p.decided))],
      h("dt", {}, "Changes"), h("dd", {}, p.changes.map((c) => `${c.path} ${c.action}`).join(", "))),
    receipt(p),
    checkSection(p.check),
    h("section", { class: "band", "aria-labelledby": "h-diff" }, h("h2", { id: "h-diff" }, "Difference"), ...p.changes.map(fileDiff)),
    p.state === "proposed" ? decision(p, passed, draw) : after(p),
  ];
}

function checkSection(check) {
  const passed = check.status === "passed";
  const section = h("section", { class: "band", "aria-labelledby": "h-check" },
    h("h2", { id: "h-check" }, "Structural check"),
    h("p", { class: `outcome ${passed ? "st-done" : "st-fail"}` }, h("span", { class: "lamp" }),
      passed ? "passed: Model check, registry, File check, execution settings and context selection accept the proposed tree" : `${check.status.replace("_", " ")}: the proposal cannot be accepted`));
  if (check.findings?.length) {
    section.append(h("ul", { class: "findings" }, check.findings.map((f) => h("li", {},
      h("span", { class: "fstage" }, f.stage), " ", h("span", { class: "digest" }, f.code), f.path && [" at ", h("span", { class: "digest" }, f.path)],
      h("div", {}, f.message)))));
  }
  return section;
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
  const section = h("section", { class: "band decide", "aria-labelledby": "h-decide" }, h("h2", { id: "h-decide" }, "Your decision"));
  const status = h("p", { class: "note", role: "status", tabindex: -1 });
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
    status.textContent = passed ? "" : "Accepting needs a passed structural check. Change the settings and propose again, or discard this proposal.";
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
  section.append(actions, status);
  return section;
}

// receipt is the publication's own outcome with each path's read-back state.
function receipt(p) {
  if (!p.receipt && !p.error) return null;
  const box = h("section", { class: "band", "aria-labelledby": "h-receipt" }, h("h2", { id: "h-receipt" }, "Publication"));
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
    return h("section", { class: "band" }, h("h2", {}, "Next"),
      h("p", {}, "The change is in the working tree, not committed. Commit it to make it the source of the next Apply and of the next proposal; until then the cockpit makes no new proposal."),
      h("p", {}, h("a", { href: "#/" }, "Back to the cockpit")));
  }
  if (p.state === "failed") {
    return h("section", { class: "band" }, h("h2", {}, "Next"),
      h("p", {}, p.receipt && ["partial", "unknown"].includes(p.receipt.outcome)
        ? "The working tree may hold part of the change. Inspect the paths above before committing or proposing again."
        : "Nothing was written. Make the proposal again from the current committed source."),
      h("p", {}, h("a", { href: "#/settings" }, "Execution settings")));
  }
  return h("p", {}, h("a", { href: "#/" }, "Back to the cockpit"));
}

// proposalList is the entry's short list of this cockpit's proposals, newest
// first.
export function proposalList(proposals) {
  if (!proposals.length) return h("p", { class: "muted" }, "No proposal yet in this cockpit.");
  return h("ul", { class: "proplist" }, proposals.map((p) => h("li", {},
    h("span", { class: `outcome ${stateClass[p.state] || ""}` }, h("span", { class: "lamp" })),
    h("a", { href: `#/proposal/${encodeURIComponent(p.id)}` }, `${p.id}: ${p.title}`),
    h("span", { class: "muted" }, ` ${stateWords[p.state] || p.state}, made ${when(p.created)}`))));
}

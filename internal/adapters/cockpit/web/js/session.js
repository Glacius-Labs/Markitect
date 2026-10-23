// The model session: the conversation on the left, the session's latest
// proposal on the right with its difference, check and the owner's decision.
// The owner's prompt goes to an Agent of Root's Execute profile in a
// container; the Agent may change only the model and the registry of its own
// copy and proposes. Every prompt is one Agent turn: the page says so before
// the first one and shows each turn's reported usage. What the Agent writes is
// shown as its text, never as a fact the cockpit established.

import { api } from "./api.js";
import { h, clear, append, clock, when, duration, announce, about, empty } from "./dom.js";
import { refusal } from "./settings.js";
import { refreshState } from "./shell.js";
import { mountProposal } from "./proposal.js";
import { store } from "./store.js";

const POLL_MS = 2000;
const stateWords = {
  preparing: "Preparing: capturing the source and starting the workspace",
  working: "Working on your prompt",
  waiting: "Waiting for your next prompt",
  halted: "Halted: takes no prompt; close it",
  closing: "Closing",
  closed: "Closed",
};
const stateClass = { preparing: "st-working", working: "st-working", waiting: "st-waiting", halted: "st-fail", closing: "st-none", closed: "st-none" };
const active = (s) => s.state === "preparing" || s.state === "working" || s.state === "closing";
let draft = ""; // the composer's text survives redraws and leaving the page

export function renderSession(content) {
  const left = h("div", { class: "pane" });
  const right = h("div", { class: "pane" });
  const split = h("div", { class: "split" }, left, right);
  content.append(split);
  const head = h("div", { class: "panehead" });
  const conversation = h("div", { class: "panebody" });
  const composer = h("section", { class: "composer", "aria-label": "Your prompt" });
  left.append(head, conversation, composer);
  const side = h("div", {});
  right.append(side);
  let timer = null;
  let shownHead = null;
  let shownComposer = null;
  let shownProposal = null;
  let stopProposal = null;
  const refresh = () => {
    clearTimeout(timer);
    api.sessions().then((list) => {
      if (!left.isConnected) return stopProposal?.();
      const current = list.sessions.find((s) => s.state !== "closed") || null;
      // The head and its close button are drawn again only when the state
      // changes, so a pending close keeps its button and its message.
      const headKey = current ? `${current.id}:${current.state}:${current.turns.length}:${current.error || ""}` : "none";
      if (headKey !== shownHead) {
        shownHead = headKey;
        drawHead(head, current, refresh);
      }
      drawConversation(conversation, list.available, current);
      const key = current ? `${current.id}:${current.state}:${current.turns.length}` : `none:${list.available}`;
      if (key !== shownComposer) {
        shownComposer = key;
        drawComposer(composer, list.available, current, refresh);
      }
      // Without a session and without earlier ones the page is one inviting
      // column; the right pane appears with a session or earlier sessions.
      split.classList.toggle("solo", !current && !list.sessions.some((x) => x.state === "closed"));
      // The right pane follows the newest proposal of the open session; with
      // none, it shows the session's facts and the closed sessions.
      const latest = current?.turns.flatMap((t) => t.proposals).at(-1) || null;
      if (latest !== shownProposal || !latest) {
        if (latest !== shownProposal) {
          stopProposal?.();
          stopProposal = null;
          shownProposal = latest;
        }
        if (latest) {
          const body = h("div", { class: "panebody" });
          clear(side).append(h("div", { class: "panehead" }, h("h2", {}, "Proposal"), h("div", { class: "actions" }, h("a", { class: "btn small", href: `#/proposal/${encodeURIComponent(latest)}` }, "Open in Decisions"))), body);
          stopProposal = mountProposal(body, latest);
        } else {
          drawSide(side, list, current);
        }
      }
      if (current && active(current)) timer = setTimeout(refresh, POLL_MS);
    }, (err) => {
      if (!left.isConnected) return;
      clear(conversation).append(h("p", { role: "alert", class: "refusal" }, refusal(err)));
      timer = setTimeout(refresh, POLL_MS * 2);
    });
  };
  refresh();
}

function drawHead(head, s, refresh) {
  clear(head).append(h("h2", {}, "Model session"));
  if (!s) return;
  head.append(h("span", { class: `outcome ${stateClass[s.state] || ""}` }, h("span", { class: "lamp" }), stateWords[s.state] || s.state));
  const actions = h("div", { class: "actions" });
  if (s.state !== "preparing") actions.append(h("span", { class: "coord", title: `Each turn at most ${s.turnTimeout}` }, `${s.turns.length}/${s.maxTurns} turns`));
  if (s.state !== "closing") actions.append(closeButton(s, refresh));
  head.append(actions);
}

function drawConversation(section, available, s) {
  clear(section);
  if (!available) {
    append(section, [empty("Model sessions are not available", "They need a run configuration and a Runtime profile."),
      store.currentProject?.saved && !store.currentProject.setUp
        ? h("p", { class: "void-action" }, h("a", { class: "btn act small", href: `#/projects?setup=${encodeURIComponent(store.currentProject.name)}` }, "Set up runs for this project"))
        : null,
      about("From the command line", h("p", {}, "Start the cockpit with markitect cockpit --run-config PATH --runtime-profile PATH --repo PATH. Sessions use the run configuration's container engine, workspace helper and credentials."))]);
    return;
  }
  if (!s) {
    const steps = ["Describe the change", "An Agent works on a copy", "It checks and proposes", "You decide"];
    section.append(h("div", { class: "intro" },
      h("p", { class: "lead" }, "Describe a change to the model. An Agent proposes it and you decide."),
      h("ol", { class: "stepper", "aria-label": "How a session goes" }, steps.map((x) => h("li", {}, x))),
      about("How it works",
        h("p", {}, "Starting a session captures the committed source with the Runtime profile."),
        h("p", {}, "An Agent of Root's Execute profile works on its own copy in a container; it may change only the model's Definitions and the registry."),
        h("p", {}, "The structural check judges its workspace; it submits a proposal with a summary, or answers you."),
        h("p", {}, "The proposal appears here with its difference and check. Accepting writes it into the working tree; nothing is committed and no Apply starts. A session's proposals start from its source: after you accept one, commit it and start a new session."))));
    return;
  }
  if (s.state === "preparing") {
    section.append(h("p", { class: "note" }, "The session's profile and limits are known once its source is prepared."));
  }
  if (s.error) section.append(h("p", { class: "refusal", role: "alert", style: { marginBottom: "18px" } }, s.error));
  const list = h("ol", { class: "conversation" });
  for (const t of s.turns) {
    const agent = h("div", { class: "agentturn" });
    for (const tool of t.tools) {
      agent.append(h("p", { class: `tooluse${tool.refused ? " st-fail" : ""}` }, h("span", { class: "faint" }, clock(tool.at)), h("b", {}, tool.name + (tool.refused ? " refused" : "")), h("span", { class: "muted" }, firstLine(tool.answer))));
    }
    for (const id of t.proposals) {
      agent.append(h("p", {}, "Proposal ", h("a", { href: `#/proposal/${encodeURIComponent(id)}` }, id), " is shown on the right."));
    }
    for (const answer of t.answers || []) agent.append(h("div", { class: "prose agenttext" }, answer));
    if (t.ended) {
      if (t.final) agent.append(h("details", {}, h("summary", { class: "note" }, "The Agent's final text"), h("div", { class: "prose agenttext" }, t.final)));
      agent.append(h("p", { class: `note ${t.stopped ? "" : "st-fail"}` }, `Turn ended ${clock(t.ended)} after ${duration(new Date(t.ended) - new Date(t.started))}; ${usageText(t.usage)}. `,
        t.stopped ? "The container's end was observed." : "The container's end was not observed.", t.reason ? ` Reported: ${t.reason}` : ""));
    } else {
      agent.append(h("p", { class: "st-working" }, `Working since ${clock(t.started)}.`));
    }
    list.append(h("li", { class: "msgpair" },
      h("div", { class: "ownerprompt" }, h("span", { class: "who label" }, "You"), h("div", { class: "prose" }, t.prompt)),
      h("div", {}, h("span", { class: "who label" }, `Agent, ${s.profile || "Root's Execute profile"}`), agent)));
  }
  section.append(list);
}

function drawSide(side, list, s) {
  const body = h("div", { class: "panebody" });
  clear(side).append(h("div", { class: "panehead" }, h("h2", {}, s ? "Session" : "Earlier sessions")), body);
  if (s && s.state !== "preparing") {
    body.append(h("dl", { class: "facts" },
      h("dt", {}, "Session"), h("dd", { class: "digest" }, s.id),
      h("dt", {}, "Source"), h("dd", {}, h("span", { class: "digest" }, (s.head || "").slice(0, 12))),
      h("dt", {}, "Agent profile"), h("dd", {}, s.profile),
      h("dt", {}, "Limits"), h("dd", {}, `${s.turns.length} of ${s.maxTurns} Agent turns used; each turn at most ${s.turnTimeout}`),
      h("dt", {}, "Reported usage"), h("dd", {}, usageText(s.usage)),
      s.evidence && [h("dt", {}, "Evidence"), h("dd", { class: "digest" }, s.evidence)],
      s.error && [h("dt", {}, "Problem"), h("dd", { class: "st-fail" }, s.error)]),
    h("p", { class: "note" }, "The Agent's proposals appear here with their difference and check as soon as it submits one."));
  }
  const closed = (list.sessions || []).filter((x) => x.state === "closed");
  if (closed.length) {
    body.append(h("h3", { class: "label sech" }, "Closed sessions"),
      h("ul", { class: "rows" }, closed.map((x) => {
        const proposals = x.turns.flatMap((t) => t.proposals);
        return h("li", {}, h("div", { class: "row static" },
          h("span", { class: "lamp st-none" }),
          h("span", { class: "title digest" }, x.id),
          h("span", { class: "meta" }, `${x.turns.length} turn${x.turns.length === 1 ? "" : "s"}`),
          h("span", { class: "sub" }, `${when(x.created)}; ${usageText(x.usage)}; proposals: `,
            proposals.length ? proposals.map((id, i) => [i ? ", " : "", h("a", { href: `#/proposal/${encodeURIComponent(id)}` }, id)]) : "none",
            x.error ? `; ${x.error}` : "")));
      })));
  }
}

function drawComposer(section, available, s, refresh) {
  clear(section);
  section.hidden = !available || (s && s.state !== "waiting" && s.state !== "working" && s.state !== "preparing");
  if (section.hidden) return;
  if (s && s.state !== "waiting") {
    section.append(h("p", { class: "note" }, "You can write the next prompt when the Agent has answered."));
    return;
  }
  const atLimit = s && s.turns.length >= s.maxTurns;
  const id = "prompt-text";
  const text = h("textarea", { id, rows: s ? 3 : 5, disabled: atLimit, placeholder: s ? "Your next prompt" : "For example: Lower the highest allowed quantity from 9999 to 5000." });
  text.value = draft;
  text.addEventListener("input", () => { draft = text.value; });
  const status = h("p", { class: "note", role: "status", tabindex: -1 });
  const actions = h("div", { class: "actions" });
  section.append(h("label", { for: id }, s ? "Your next prompt" : "What should change in the model?"), text, actions, status);
  if (atLimit) status.textContent = "The session used all its Agent turns. Close it and start a new one.";
  const send = async (button) => {
    button.disabled = true;
    status.textContent = s ? "Sending…" : "Capturing the committed source, preparing it with the Runtime profile and starting the container. This takes a while.";
    try {
      const result = s ? await api.promptSession(s.id, text.value) : await api.startSession(text.value);
      draft = "";
      announce(`Session ${result.id}: ${stateWords[result.state] || result.state}.`);
      refreshState();
      refresh();
    } catch (err) {
      button.disabled = false;
      status.textContent = `Not sent: ${refusal(err)}`;
      status.focus();
    }
  };
  const ready = () => {
    if (!text.value.trim()) {
      status.textContent = "Write the prompt first.";
      text.focus();
      return false;
    }
    return true;
  };
  if (s) {
    const button = h("button", { class: "btn act", type: "button", disabled: atLimit }, "Send the prompt");
    button.addEventListener("click", () => ready() && send(button));
    text.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && !button.disabled && ready()) send(button);
    });
    actions.append(button, h("span", { class: "note" }, "One Agent turn; Ctrl+Enter sends."));
    return;
  }
  // Starting is a deliberate two-step action: it starts provider work.
  const review = h("button", { class: "btn act", type: "button" }, "Start a session");
  review.addEventListener("click", () => {
    if (!ready()) return;
    text.readOnly = true;
    const start = h("button", { class: "btn act", type: "button" }, "Start the session and send the prompt");
    const back = h("button", { class: "btn", type: "button" }, "Edit");
    back.addEventListener("click", () => {
      text.readOnly = false;
      clear(actions).append(review, h("span", { class: "note" }, "Each prompt is one Agent turn and costs provider usage."));
      status.textContent = "";
      text.focus();
    });
    start.addEventListener("click", () => send(start));
    clear(actions).append(start, back);
    status.textContent = "This starts a container with an Agent of Root's Execute profile for one Agent turn, which costs provider usage. Only a proposal you accept writes anything.";
    start.focus();
  });
  actions.append(review, h("span", { class: "note" }, "Each prompt is one Agent turn and costs provider usage."));
}

function closeButton(s, refresh) {
  const button = h("button", { class: "btn small", type: "button" }, s.state === "working" || s.state === "preparing" ? "Stop and close" : "Close the session");
  button.addEventListener("click", async () => {
    button.disabled = true;
    try {
      const closed = await api.closeSession(s.id);
      announce(closed.error ? `Session ${s.id} closed with a problem: ${closed.error}` : `Session ${s.id} closed.`);
    } catch (err) {
      button.disabled = false;
      button.after(h("span", { class: "note st-fail", role: "alert" }, ` Not closed: ${refusal(err)}`));
      return;
    }
    refreshState();
    refresh();
  });
  return button;
}

// usageText states what the provider reported; a zero report is not shown as
// free.
function usageText(u) {
  if (!u || !u.providerTurns && !u.inputTokens && !u.outputTokens) return "no usage reported";
  const cached = (u.cacheReadTokens || 0) + (u.cacheCreationTokens || 0);
  const input = (u.inputTokens || 0) + cached;
  const cost = u.costUSD ? `, ${u.costUSD.toFixed(2)} USD estimated by the provider` : "";
  return `${input.toLocaleString()} input tokens${u.cacheReadTokens ? ` (${u.cacheReadTokens.toLocaleString()} read from cache)` : ""}, ${(u.outputTokens || 0).toLocaleString()} output tokens${cost}`;
}

function firstLine(text) {
  const line = (text || "").split("\n")[0];
  return line.length > 200 ? `${line.slice(0, 199)}…` : line;
}

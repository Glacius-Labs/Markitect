// The model session page. The owner's prompt goes to an Agent of Root's
// Execute profile in a container; the Agent may change only the model and the
// registry of its own copy and proposes, and the owner decides on each
// proposal on its own page. Every prompt is one provider turn: the page says
// so before the first one and shows each turn's reported usage. What the Agent
// writes is shown as its text, never as a fact the cockpit established.

import { api } from "./api.js";
import { h, clear, clock, duration, announce } from "./dom.js";
import { top, refusal } from "./settings.js";

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

export function renderModel(app) {
  document.title = "Model session – Markitect cockpit";
  clear(app);
  const main = h("main", { id: "main", class: "runs authoring" });
  app.append(main);
  const head = h("div", {});
  const conversation = h("section", { class: "band", "aria-labelledby": "h-conversation" });
  const composer = h("section", { class: "band composer", "aria-label": "Your prompt" });
  const past = h("section", { class: "band", "aria-labelledby": "h-past" });
  main.append(top("Model session"), head, conversation, composer, past);
  let timer = null;
  let shown = null; // the session and state the composer was drawn for
  const refresh = () => {
    clearTimeout(timer);
    api.sessions().then((list) => {
      if (!main.isConnected) return;
      const open = list.sessions.find((s) => s.state !== "closed");
      const current = open || null;
      drawHead(head, list.available, current);
      drawConversation(conversation, current);
      const key = current ? `${current.id}:${current.state}:${current.turns.length}` : `none:${list.available}`;
      if (key !== shown) {
        shown = key;
        drawComposer(composer, list.available, current, refresh);
      }
      drawPast(past, list.sessions.filter((s) => s.state === "closed"));
      if (current && active(current)) timer = setTimeout(refresh, POLL_MS);
    }, (err) => {
      if (!main.isConnected) return;
      clear(head).append(h("p", { role: "alert", class: "refusal" }, refusal(err)));
      timer = setTimeout(refresh, POLL_MS * 2);
    });
  };
  refresh();
}

function drawHead(head, available, s) {
  clear(head);
  if (!available) {
    head.append(h("p", { class: "lead" }, "Model sessions need the cockpit to be started with a run configuration and a Runtime profile: markitect cockpit --run-config PATH --runtime-profile PATH --repo PATH. They use the run configuration's container engine, workspace helper and credentials."));
    return;
  }
  head.append(h("p", { class: "lead" }, "Your prompt goes to an Agent of Root's Execute profile in a container. It works on its own copy of the committed source, may change only .markitect/model and .markitect/registry, and proposes; you decide on every proposal. Each prompt is one Agent turn, which costs provider usage. A session's proposals start from its source: after you accept one, commit it and start a new session."));
  if (!s) return;
  const used = s.turns.length;
  if (s.state === "preparing") {
    head.append(h("p", { class: `outcome big ${stateClass.preparing}`, tabindex: -1 }, h("span", { class: "lamp" }), stateWords.preparing),
      h("p", { class: "note" }, "The session's profile and limits are known once its source is prepared."));
    return;
  }
  head.append(h("p", { class: `outcome big ${stateClass[s.state] || ""}`, tabindex: -1 }, h("span", { class: "lamp" }), stateWords[s.state] || s.state),
    h("dl", { class: "facts" },
      h("dt", {}, "Session"), h("dd", {}, h("span", { class: "digest" }, s.id), ` from the committed source at `, h("span", { class: "digest" }, (s.head || "").slice(0, 12))),
      h("dt", {}, "Agent profile"), h("dd", {}, s.profile || "not yet known"),
      h("dt", {}, "Limits"), h("dd", {}, `${used} of ${s.maxTurns} Agent turns used; each turn at most ${s.turnTimeout}`),
      h("dt", {}, "Reported usage"), h("dd", {}, usageText(s.usage)),
      s.evidence && [h("dt", {}, "Evidence"), h("dd", { class: "digest" }, s.evidence)],
      s.error && [h("dt", {}, "Problem"), h("dd", { class: "st-fail" }, s.error)]));
}

function drawConversation(section, s) {
  clear(section);
  if (!s) return;
  section.append(h("h2", { id: "h-conversation" }, "Conversation"));
  const list = h("ol", { class: "conversation" });
  for (const t of s.turns) {
    const agent = h("div", { class: "agentturn" });
    for (const tool of t.tools) {
      agent.append(h("p", { class: `tooluse${tool.refused ? " st-fail" : ""}` }, `${clock(tool.at)} ${tool.name}${tool.refused ? " refused" : ""}: `, h("span", { class: "muted" }, firstLine(tool.answer))));
    }
    for (const id of t.proposals) {
      agent.append(h("p", {}, "Proposal ", h("a", { href: `#/proposal/${encodeURIComponent(id)}` }, id), ": review and decide on its own page."));
    }
    for (const answer of t.answers || []) agent.append(h("div", { class: "prose agent" }, answer));
    if (t.ended) {
      if (t.final) agent.append(h("details", {}, h("summary", {}, "The Agent's final text"), h("div", { class: "prose agent" }, t.final)));
      agent.append(h("p", { class: t.stopped ? "faint" : "st-fail" }, `Turn ended ${clock(t.ended)} after ${duration(new Date(t.ended) - new Date(t.started))}; ${usageText(t.usage)}. `,
        t.stopped ? "The container's end was observed." : "The container's end was not observed.", t.reason ? ` Reported: ${t.reason}` : ""));
    } else {
      agent.append(h("p", { class: "st-working" }, `Working since ${clock(t.started)}.`));
    }
    list.append(h("li", { class: "exchange" },
      h("div", { class: "ownerprompt" }, h("span", { class: "who" }, "You"), h("div", { class: "prose" }, t.prompt)),
      h("div", {}, h("span", { class: "who" }, "Agent"), agent)));
  }
  section.append(list);
}

function drawComposer(section, available, s, refresh) {
  clear(section);
  if (!available || (s && s.state !== "waiting")) {
    if (s && (s.state === "working" || s.state === "preparing")) section.append(h("p", { class: "note" }, "You can write the next prompt when the Agent has answered."));
    if (s && s.state !== "closing") section.append(closeButton(s, refresh));
    return;
  }
  const atLimit = s && s.turns.length >= s.maxTurns;
  const id = "prompt-text";
  const text = h("textarea", { id, rows: 5, disabled: atLimit });
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
    actions.append(button, closeButton(s, refresh));
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
      clear(actions).append(review);
      status.textContent = "";
      text.focus();
    });
    start.addEventListener("click", () => send(start));
    clear(actions).append(start, back);
    status.textContent = "This starts a container with an Agent of Root's Execute profile for one Agent turn, which costs provider usage. Only a proposal you accept writes anything.";
    start.focus();
  });
  actions.append(review);
}

function closeButton(s, refresh) {
  const button = h("button", { class: "btn", type: "button" }, s.state === "working" || s.state === "preparing" ? "Stop and close the session" : "Close the session");
  button.addEventListener("click", async () => {
    button.disabled = true;
    try {
      const closed = await api.closeSession(s.id);
      announce(closed.error ? `Session ${s.id} closed with a problem: ${closed.error}` : `Session ${s.id} closed.`);
    } catch (err) {
      button.disabled = false;
      button.after(h("p", { class: "note", role: "alert" }, `Not closed: ${refusal(err)}`));
      return;
    }
    refresh();
  });
  return button;
}

function drawPast(section, sessions) {
  clear(section);
  if (!sessions.length) return;
  section.append(h("h2", { id: "h-past" }, "Closed sessions"),
    h("ul", { class: "proplist" }, sessions.map((s) => {
      const proposals = s.turns.flatMap((t) => t.proposals);
      return h("li", {},
        h("span", { class: "digest" }, s.id),
        h("span", { class: "muted" }, ` ${s.turns.length} turn${s.turns.length === 1 ? "" : "s"}, ${usageText(s.usage)}; proposals: `),
        proposals.length ? proposals.map((id, i) => [i ? ", " : "", h("a", { href: `#/proposal/${encodeURIComponent(id)}` }, id)]) : h("span", { class: "muted" }, "none"),
        s.error && h("span", { class: "st-fail" }, `; ${s.error}`));
    })));
}

// usageText states what the provider reported; a zero report is not shown as
// free.
function usageText(u) {
  if (!u || !u.providerTurns && !u.inputTokens && !u.outputTokens) return "no usage reported";
  const cost = u.costUSD ? `, ${u.costUSD.toFixed(2)} USD estimated by the provider` : "";
  return `${u.inputTokens} input and ${u.outputTokens} output tokens${cost}`;
}

function firstLine(text) {
  const line = (text || "").split("\n")[0];
  return line.length > 200 ? `${line.slice(0, 199)}…` : line;
}

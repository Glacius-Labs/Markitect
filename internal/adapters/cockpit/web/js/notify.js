// What waits for the owner, outside the page: desktop notifications and the
// count in the window title. Both follow the shared observation: each new
// observation is compared with the previous one, and only a change notifies.
// The first observation after the page loads is the baseline and notifies
// nothing. Notifications need the owner's permission, asked only from the
// bell in the top bar; without it everything else works as before and the
// browser keeps the permission, not the cockpit. Nothing here writes.

import { h, areaIcon } from "./dom.js";
import { store } from "./store.js";

const WORKING = new Set(["working", "preparing"]);

// snapshot reduces an observation to what notifications compare, one entry
// per part of the observation. A part that has not arrived yet (its read
// failed so far) is null, and nothing is compared for it.
export function snapshot(s) {
  const runs = s.data.runs ? { live: new Set(s.liveRuns.map((r) => r.id)), byId: new Map(s.data.runs.map((r) => [r.id, r])) } : null;
  const session = s.data.sessions ? s.openSession : null;
  return {
    runs,
    sessions: s.data.sessions ? { open: session ? { id: session.id, working: WORKING.has(session.state), state: session.state } : null } : null,
    proposals: s.data.proposals ? new Set(s.proposalsToDecide.map((p) => p.id)) : null,
  };
}

// merge keeps, per part, the last observation that arrived: a part that
// failed to load in a poll keeps its earlier value, and a part that arrives
// for the first time becomes its baseline without notifying.
export function merge(prev, next) {
  return {
    runs: next.runs || prev?.runs || null,
    sessions: next.sessions || prev?.sessions || null,
    proposals: next.proposals || prev?.proposals || null,
  };
}

// changes lists the notifications a new observation causes against the
// previous one. Every item has a title, a body and the place a click opens.
// Only a part present in both observations is compared.
export function changes(prev, next) {
  const out = [];
  if (prev.runs && next.runs) {
    for (const id of prev.runs.live) {
      if (next.runs.live.has(id)) continue;
      const run = next.runs.byId.get(id);
      const outcome = run?.runOutcome || run?.outcome;
      out.push({ tag: `run-ended-${id}`, title: `Run ${id} ended`, body: outcome ? `Outcome: ${outcome}.` : "Open it to see how it ended.", href: `#/run/${encodeURIComponent(id)}` });
    }
  }
  const was = prev.sessions?.open;
  const now = next.sessions?.open;
  if (was?.working && now && now.id === was.id) {
    if (now.state === "waiting") out.push({ tag: `session-${now.id}`, title: "The model session answered", body: "The session waits for your next prompt.", href: "#/session" });
    else if (now.state === "halted") out.push({ tag: `session-${now.id}`, title: "The model session halted", body: "It takes no more prompts; open it to close it.", href: "#/session" });
  }
  if (prev.proposals && next.proposals) {
    for (const id of next.proposals) {
      if (prev.proposals.has(id)) continue;
      out.push({ tag: `proposal-${id}`, title: "A proposal waits for your decision", body: `Proposal ${id}.`, href: `#/proposal/${encodeURIComponent(id)}` });
    }
  }
  return out;
}

// pending is how many things wait for the owner's decision.
export function pending(s) {
  return s.proposalsToDecide.length;
}

const supported = typeof Notification !== "undefined";
let previous = null; // per part, the last observation that arrived
let baseTitle = document.title;
let bell = null;

function permission() {
  return supported ? Notification.permission : "denied";
}

function notify(item) {
  if (permission() !== "granted") return;
  // The page shows it already while the owner looks at it.
  if (document.visibilityState === "visible" && document.hasFocus()) return;
  try {
    const n = new Notification(item.title, { body: item.body, tag: item.tag });
    n.onclick = () => {
      window.focus();
      location.hash = item.href;
      n.close();
    };
  } catch (err) {
    console.error(err); // best effort: a browser that refuses changes nothing else
  }
}

// setTitle names the place; the count of what waits is prefixed here only.
export function setTitle(text) {
  baseTitle = text;
  drawTitle();
}

function drawTitle() {
  const n = pending(store);
  document.title = `${n ? `(${n}) ` : ""}${baseTitle}`;
}

function observe(s) {
  const next = snapshot(s);
  // A part that failed to arrive leaves its last value, so a failed poll is
  // not a change and the next successful one does not burst.
  if (previous) for (const item of changes(previous, next)) notify(item);
  previous = merge(previous, next);
  drawTitle();
  drawBell();
}

const bellState = {
  default: { label: "Notifications off", title: "Notifications off. Click to ask the browser to allow desktop notifications for ended runs, model session answers and proposals.", lamp: "st-none" },
  granted: { label: "Notifications on", title: "Notifications on. The browser lets the cockpit notify you; change this in the browser's site settings.", lamp: "st-done" },
  denied: { label: "Notifications blocked", title: "Notifications blocked. The browser blocks them for the cockpit; allow them in the browser's site settings.", lamp: "st-fail" },
};

// The bell is an icon with a status dot; its name and tooltip say the state.
function drawBell() {
  if (!bell) return;
  const state = bellState[permission()] || bellState.default;
  // Blocked stays focusable and says why; a click does nothing.
  if (permission() === "denied") bell.setAttribute("aria-disabled", "true");
  else bell.removeAttribute("aria-disabled");
  bell.title = state.title;
  bell.setAttribute("aria-label", state.label);
  bell.className = `btn quiet icon bell${permission() === "denied" ? " blocked" : permission() === "granted" ? " on" : ""}`;
  bell.replaceChildren(areaIcon("bell"), h("span", { class: `lamp ${state.lamp}` }));
}

// notifyControl is the bell in the top bar: it shows whether notifications
// are on, off or blocked and, while the browser has not been asked, asks on
// the owner's click. It is a status, not a switch: the page cannot turn
// notifications off, the browser's site settings can. It is absent where
// the browser has no notifications.
export function notifyControl() {
  if (!supported || bell) return null;
  bell = h("button", {
    type: "button",
    onclick: async () => {
      if (permission() !== "default") return;
      try {
        await Notification.requestPermission();
      } catch (err) {
        console.error(err);
      }
      drawBell();
    },
  });
  drawBell();
  return bell;
}

store.subscribe(observe);

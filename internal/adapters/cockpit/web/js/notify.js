// What waits for the owner, outside the page: desktop notifications and the
// count in the window title. Both follow the shared observation: each new
// observation is compared with the previous one, and only a change notifies.
// The first observation after the page loads is the baseline and notifies
// nothing. Notifications need the owner's permission, asked only from the
// bell in the top bar; without it everything else works as before and the
// browser keeps the permission, not the cockpit. Nothing here writes.

import { h } from "./dom.js";
import { store } from "./store.js";

const WORKING = new Set(["working", "preparing"]);

// snapshot reduces an observation to what notifications compare.
export function snapshot(s) {
  const runs = new Map((s.data.runs || []).map((r) => [r.id, r]));
  const session = s.openSession;
  return {
    live: new Set(s.liveRuns.map((r) => r.id)),
    runs,
    session: session ? { id: session.id, working: WORKING.has(session.state), waiting: session.state === "waiting" } : null,
    proposals: new Set(s.proposalsToDecide.map((p) => p.id)),
  };
}

// changes lists the notifications a new observation causes against the
// previous one. Every item has a title, a body and the place a click opens.
export function changes(prev, next) {
  const out = [];
  for (const id of prev.live) {
    if (next.live.has(id)) continue;
    const run = next.runs.get(id);
    const outcome = run?.runOutcome || run?.outcome;
    out.push({ tag: `run-ended-${id}`, title: `Run ${id} ended`, body: outcome ? `Outcome: ${outcome}.` : "Open it to see how it ended.", href: `#/run/${encodeURIComponent(id)}` });
  }
  if (prev.session?.working && next.session && next.session.id === prev.session.id && next.session.waiting) {
    out.push({ tag: `session-${next.session.id}`, title: "The model session answered", body: "The session waits for your next prompt.", href: "#/session" });
  }
  for (const id of next.proposals) {
    if (prev.proposals.has(id)) continue;
    out.push({ tag: `proposal-${id}`, title: "A proposal waits for your decision", body: `Proposal ${id}.`, href: `#/proposal/${encodeURIComponent(id)}` });
  }
  return out;
}

// pending is how many things wait for the owner's decision.
export function pending(s) {
  return s.proposalsToDecide.length;
}

const supported = typeof Notification !== "undefined";
let previous = null;
let baseTitle = document.title;
let bell = null;

function permission() {
  return supported ? Notification.permission : "denied";
}

function notify(item) {
  if (permission() !== "granted") return;
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
  // Parts that failed to arrive leave their last value; a failed poll is not
  // a change.
  if (previous) for (const item of changes(previous, next)) notify(item);
  previous = next;
  drawTitle();
  drawBell();
}

const bellText = {
  default: { label: "Notifications off", title: "Turn on desktop notifications for ended runs, model session answers and proposals", on: false },
  granted: { label: "Notifications on", title: "The browser lets the cockpit notify you; change this in the site settings of the browser", on: true },
  denied: { label: "Notifications blocked", title: "The browser blocks notifications for the cockpit; allow them in the site settings of the browser", on: false },
};

function drawBell() {
  if (!bell) return;
  const state = bellText[permission()] || bellText.default;
  bell.setAttribute("aria-pressed", state.on ? "true" : "false");
  bell.disabled = permission() === "denied";
  bell.title = state.title;
  bell.className = `btn quiet small bell${permission() === "denied" ? " blocked" : ""}`;
  bell.replaceChildren(h("span", { class: `lamp ${state.on ? "st-done" : permission() === "denied" ? "st-fail" : "st-none"}` }), h("span", { class: "belltext" }, state.label));
}

// notifyControl is the bell in the top bar: it shows whether notifications
// are on, off or blocked and, while the browser has not been asked, asks on
// the owner's click. It is absent where the browser has no notifications.
export function notifyControl() {
  if (!supported || bell) return null;
  bell = h("button", {
    type: "button",
    onclick: async () => {
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

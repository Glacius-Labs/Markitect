// Starting and stopping the one run this cockpit runs in its own process.
// Starting takes two steps: the confirmation names the committed execution
// budget, the run configuration's deadline and the Execute Agent profile with
// its model, and says that a run spends provider usage. Stopping takes two
// steps as well and ends the run incomplete, keeping its evidence and report.
// Only the run this cockpit started can be stopped here; runs started
// elsewhere are observed in the list. The shared state lives in this module so
// the card in the Runs area and the stop control in the inspector agree, and
// every widget is drawn again only when what it shows changes, so focus
// survives the four-second polls.

import { api } from "./api.js";
import { h, clear, append, when, announce } from "./dom.js";
import { store } from "./store.js";
import { refusal } from "./settings.js";
import { refreshState } from "./shell.js";

const ui = {
  step: "idle", // idle | confirm
  facts: null,
  factsError: "",
  loading: false,
  busy: false,
  message: "",
  stopStep: "idle", // idle | confirm
  stopBusy: false,
  stopMessage: "",
  focus: false, // move focus to the primary button after the next draw
  focusHeading: false, // move focus to the card's heading after the next draw
};
const mounted = new Set();
let watching = false;
let lastState = null;

const stateWords = {
  starting: "Starting: capturing the committed source and preparing the run",
  running: "Running",
  stopping: "Stopping: the run ends incomplete and writes its report",
};
const outcomeClass = { success: "st-done", failed: "st-fail", incomplete: "st-waiting", refused: "st-fail" };

function signature() {
  if (!store.runStarted) {
    ui.stopStep = "idle";
    ui.stopBusy = false;
    ui.stopMessage = "";
  }
  return JSON.stringify([store.data.runControl, store.errors.runControl?.message, ui]);
}

function notify() {
  for (const draw of [...mounted]) draw();
}

// watch announces the end of the run this cockpit started; the polls reach
// every mounted widget through the store.
function watch() {
  if (watching) return;
  watching = true;
  store.subscribe(() => {
    const run = store.data.runControl?.run;
    const state = run?.state || null;
    if (lastState && lastState !== "ended" && state === "ended") {
      announce(run.outcome === "refused" ? `The run was refused: ${run.error}` : `The run ended: ${run.outcome}.`);
    }
    lastState = state;
    notify();
  });
}

function mount(el, render) {
  let shown = null;
  let seen = false;
  const draw = () => {
    if (el.isConnected) seen = true;
    else if (seen) {
      mounted.delete(draw);
      return;
    }
    const sig = signature();
    if (sig === shown) return;
    shown = sig;
    // A redraw because the run changed keeps the focus on the same button.
    const focused = el.contains(document.activeElement) && document.activeElement !== el ? document.activeElement.textContent : null;
    render(clear(el));
    if (focused !== null) [...el.querySelectorAll("button")].find((b) => b.textContent === focused)?.focus();
    if (ui.focusHeading) {
      const heading = el.querySelector("h3[tabindex]");
      if (heading) {
        ui.focusHeading = false;
        heading.focus();
      }
    }
    if (ui.focus) {
      const primary = el.querySelector("[data-primary]");
      if (primary) {
        ui.focus = false;
        primary.focus();
      }
    }
  };
  mounted.add(draw);
  watch();
  draw();
  return el;
}

// ---------- actions ----------

function openConfirmation() {
  ui.step = "confirm";
  ui.facts = null;
  ui.factsError = "";
  ui.message = "";
  ui.loading = true;
  notify();
  api.runControl(true).then((view) => {
    ui.facts = view.facts || null;
    ui.factsError = view.factsError || "";
  }, (err) => {
    ui.factsError = err.message || String(err);
  }).finally(() => {
    ui.loading = false;
    ui.focus = true;
    notify();
  });
}

function cancelStart() {
  ui.step = "idle";
  ui.message = "";
  notify();
}

async function startRun() {
  ui.busy = true;
  ui.message = "";
  notify();
  let changed = false;
  try {
    await api.startRun(ui.facts.head);
    await refreshState();
    ui.step = "idle";
    ui.focusHeading = true;
    announce("The run is starting.");
  } catch (err) {
    ui.message = `Not started: ${refusal(err)}`;
    changed = /changed since the confirmation/.test(err.message || "");
  }
  ui.busy = false;
  if (changed) {
    // The confirmation is stale: read what a run would use now.
    const message = ui.message;
    openConfirmation();
    ui.message = message;
  }
  notify();
}

function openStop() {
  ui.stopStep = "confirm";
  ui.stopMessage = "";
  ui.focus = true;
  notify();
}

function cancelStop() {
  ui.stopStep = "idle";
  ui.stopMessage = "";
  notify();
}

async function stopRun() {
  ui.stopBusy = true;
  ui.stopMessage = "";
  notify();
  try {
    await api.stopRun();
    await refreshState();
    ui.stopStep = "idle";
    announce("The run is stopping.");
  } catch (err) {
    ui.stopMessage = `Not stopped: ${refusal(err)}`;
  }
  ui.stopBusy = false;
  notify();
}

// ---------- widgets ----------

// runControlCard is the card of the Runs area: the start of a run, the run
// this cockpit started with its stop, or why starting is unavailable.
export function runControlCard() {
  const card = h("section", { class: "panel runctl", "aria-labelledby": "run-start" });
  return mount(card, (el) => {
    const rc = store.data.runControl;
    const run = store.runStarted;
    el.append(h("header", {}, h("h3", { id: "run-start", tabindex: "-1" }, run ? "Run started here" : "Start a run")));
    const body = h("div", { class: "body" });
    el.append(body);
    if (!rc) {
      body.append(store.errors.runControl
        ? h("p", { class: "void", role: "alert" }, `Starting is not readable: ${store.errors.runControl.message}`)
        : h("p", { class: "note" }, "Reading what the cockpit can start…"));
      return;
    }
    if (!rc.available) {
      body.append(h("p", { class: "note" }, rc.reason),
        h("p", { class: "note" }, "An Apply started with markitect apply still appears below while it runs; it is observed, not controlled, here."));
      return;
    }
    if (run) {
      drawActive(body, run);
      return;
    }
    if (rc.run) body.append(lastRun(rc.run));
    if (ui.step === "confirm") drawConfirmation(body);
    else {
      body.append(h("p", { class: "note" }, "Runs the committed model with the cockpit's run configuration and Runtime profile, as markitect apply does."),
        h("div", { class: "actions" }, h("button", { class: "btn act", type: "button", onclick: openConfirmation }, "Start a run…")));
    }
  });
}

function drawActive(body, run) {
  const link = run.run
    ? h("a", { href: `#/run/${encodeURIComponent(run.run)}` }, run.run)
    : h("span", { class: "note" }, "it names itself once it holds the state directory");
  body.append(
    h("p", {}, h("span", { class: "outcome st-working" }, h("span", { class: "lamp" }), stateWords[run.state] || run.state)),
    h("dl", { class: "facts" },
      h("dt", {}, "Run"), h("dd", {}, link),
      h("dt", {}, "Started"), h("dd", {}, when(run.started))),
    h("p", { class: "note" }, "Ending the cockpit stops this run the same way and waits for its report; closing the window does not."));
  if (run.state === "stopping") return;
  drawStop(body);
}

function drawStop(parent) {
  if (ui.stopStep !== "confirm") {
    parent.append(h("div", { class: "actions" }, h("button", { class: "btn small", type: "button", onclick: openStop }, "Stop the run…")));
    return;
  }
  append(parent, [
    h("p", { class: "note", role: "status" }, "Stopping ends the run incomplete. Its evidence and report are kept; nothing is restored."),
    ui.stopMessage ? h("p", { class: "refusal", role: "alert" }, ui.stopMessage) : null,
    h("div", { class: "actions" },
      h("button", { class: "btn act", type: "button", "data-primary": "", disabled: ui.stopBusy, onclick: stopRun }, ui.stopBusy ? "Stopping…" : "Stop the run"),
      h("button", { class: "btn", type: "button", disabled: ui.stopBusy, onclick: cancelStop }, "Keep it running"))]);
}

// profileRows names Root's Execute profile with its model and, when other
// Managers use other Execute profiles, each of those.
function profileRows(profiles) {
  const model = (p) => (p.model ? ` with model ${p.model}` : ", whose Runtime profile states no model");
  const [root, ...others] = profiles;
  const rows = [h("dt", {}, "Root's Execute profile"), h("dd", {}, root ? [root.name, h("span", { class: "note" }, model(root))] : "none resolved")];
  if (others.length) {
    rows.push(h("dt", {}, "Other Execute profiles"), h("dd", {}, others.map((p) => h("div", {}, p.name,
      h("span", { class: "note" }, `${model(p)}; for ${p.managers.join(", ")}`)))));
  }
  return rows;
}

function lastRun(run) {
  const cls = outcomeClass[run.outcome] || "st-none";
  if (run.outcome === "refused") {
    return h("p", { class: "refusal", role: "status" }, `The last start was refused: ${run.error}`);
  }
  return h("div", {},
    h("p", {}, h("span", { class: `outcome ${cls}` }, h("span", { class: "lamp" }), `Last run started here ended ${run.outcome || "without an outcome"}`),
      " ", run.run ? h("a", { href: `#/run/${encodeURIComponent(run.run)}` }, run.run) : null),
    run.error ? h("p", { class: "note" }, run.error) : null);
}

function drawConfirmation(body) {
  const box = h("div", { class: "confirm", role: "group", "aria-label": "Confirm the start of a run" });
  body.append(box);
  if (ui.loading) {
    box.append(h("p", { class: "note", role: "status" }, "Reading what a run would use from the committed source…"));
    return;
  }
  const cancel = h("button", { class: "btn", type: "button", disabled: ui.busy, onclick: cancelStart }, "Cancel");
  if (ui.factsError || !ui.facts) {
    box.append(h("p", { class: "refusal", role: "alert" }, `No run can start now: ${refusal(new Error(ui.factsError || "the facts are not readable"))}`),
      h("div", { class: "actions" }, h("button", { class: "btn", type: "button", "data-primary": "", onclick: openConfirmation }, "Read again"), cancel));
    return;
  }
  const f = ui.facts;
  append(box, [
    h("h4", {}, "This run will use"),
    h("dl", { class: "facts" },
      h("dt", {}, "Committed source"), h("dd", { class: "digest", title: f.head }, (f.head || "").slice(0, 12)),
      h("dt", {}, "Execution time"), h("dd", {}, `up to ${f.budget.executionTime}`),
      h("dt", {}, "Turn timeout"), h("dd", {}, f.budget.turnTimeout),
      h("dt", {}, "Agent turns"), h("dd", {}, String(f.budget.agentTurns)),
      h("dt", {}, "Passes"), h("dd", {}, `up to ${f.budget.passes}`),
      h("dt", {}, "Deadline"), h("dd", {}, f.deadline ? (when(f.deadline) || f.deadline) : "none in the run configuration"),
      ...profileRows(f.profiles || [])),
    h("p", { class: "note" }, "Starting runs the same Apply as markitect apply with the same checks, budget and publication. It spends provider usage, up to this budget. Ending the cockpit stops the run; closing the window does not."),
    ui.message ? h("p", { class: "refusal", role: "alert" }, ui.message) : null,
    h("div", { class: "actions" },
      h("button", { class: "btn act", type: "button", "data-primary": "", disabled: ui.busy, onclick: startRun }, ui.busy ? "Starting…" : "Start the run"),
      cancel)]);
}

// stopControl is the stop of the inspector's head. It shows only while the
// inspected run is the one this cockpit started and active.
export function stopControl(runId) {
  const el = h("span", { class: "runstop" });
  return mount(el, (el) => {
    const run = store.runStarted;
    el.hidden = !run || run.run !== runId;
    if (el.hidden) return;
    if (run.state === "stopping") {
      el.append(h("span", { class: "note", role: "status" }, "Stopping; the run writes its report."));
      return;
    }
    if (ui.stopStep !== "confirm") {
      el.append(h("button", { class: "btn small", type: "button", onclick: openStop }, "Stop the run…"));
      return;
    }
    append(el, [
      h("span", { class: "note", role: "status" }, "Ends the run incomplete; its evidence and report are kept; nothing is restored."),
      ui.stopMessage ? h("span", { class: "st-fail", role: "alert" }, ui.stopMessage) : null,
      h("button", { class: "btn act small", type: "button", "data-primary": "", disabled: ui.stopBusy, onclick: stopRun }, ui.stopBusy ? "Stopping…" : "Stop the run"),
      h("button", { class: "btn small", type: "button", disabled: ui.stopBusy, onclick: cancelStop }, "Keep it running")]);
  });
}

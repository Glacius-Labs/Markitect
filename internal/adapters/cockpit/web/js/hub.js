// The hub, where the owner starts: the project as its last observation saw
// it, the one next step that needs the owner, and the areas with their live
// state. An area is entered with a click or its digit. The hub writes nothing
// except asking for a new observation of the project.

import { api } from "./api.js";
import { h, clear, append, when, shortDigest, areaIcon } from "./dom.js";
import { store } from "./store.js";
import { areas, emblem, refreshState, sessionClass } from "./shell.js";

const outcomeClass = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" };

export function renderHub(content) {
  const hero = h("section", { class: "card hero", "aria-labelledby": "hub-project" });
  const tiles = new Map(areas.map((a) => [a.id, tile(a)]));
  const page = h("div", { class: "hub" },
    hero,
    h("nav", { class: "tiles", "aria-label": "Areas" }, [...tiles.values()].map((t) => t.el)),
    h("p", { class: "hint" }, h("kbd", {}, "1"), "–", h("kbd", {}, String(areas.length)), " enter an area · ", h("kbd", {}, "Esc"), " comes back"));
  content.append(page);
  const parts = { head: h("div", {}), side: h("div", { class: "side" }), next: h("div", { class: "next", role: "status" }) };
  hero.append(parts.head, parts.side, parts.next);
  // Tiles are filled in place and the hero's parts are drawn again only when
  // what they show changes, so focus and announcements survive a poll.
  const drawn = {};
  const once = (key, value, fn) => {
    const sig = JSON.stringify(value);
    if (drawn[key] === sig) return;
    drawn[key] = sig;
    fn();
  };
  const draw = (s) => {
    if (!page.isConnected) return unsubscribe();
    const part = s.data.project;
    once("project", [part?.configured, part?.observing, part?.project, s.errors.project?.message], () => drawProject(parts, s));
    const next = nextStep(s);
    once("next", next, () => drawNext(parts.next, next));
    drawModel(tiles.get("model"), s);
    drawRuns(tiles.get("runs"), s);
    drawDecisions(tiles.get("decisions"), s);
    drawSettings(tiles.get("settings"), s);
  };
  const unsubscribe = store.subscribe(draw);
  if (Object.keys(store.data).length) draw(store);
  else refreshState();
}

// tile is an area's card: emblem, digit, name and purpose, then the parts the
// live state fills: a badge, figures and a footer line.
function tile(a) {
  const badge = h("span", {});
  const stats = h("div", { class: "stats" });
  const foot = h("span", { class: "txt" });
  const footLamp = h("span", {});
  const el = h("a", { class: `tile t-${a.id}`, href: a.href, "aria-keyshortcuts": a.key },
    h("span", { class: "ghost", "aria-hidden": "true" }, areaIcon(a.id)),
    h("div", { class: "top" }, emblem(a.id), h("span", { class: "badge" }, badge, h("kbd", { title: `Press ${a.key}` }, a.key))),
    h("div", { class: "name" }, h("h2", {}, a.label), h("p", { class: "lead" }, a.lead)),
    stats,
    h("div", { class: "foot" }, footLamp, foot, h("span", { class: "go", "aria-hidden": "true" }, "Enter →")));
  return { el, badge, stats, foot, footLamp, chips: a.chips || [] };
}

// A tile without figures names what the area holds, so its middle is never
// empty and nothing moves when the figures arrive.
function fill(t, { stats = [], lamp = null, foot = "", count = 0, attention = false }) {
  t.stats.classList.toggle("chips", !stats.length);
  append(clear(t.stats), stats.length
    ? stats.map(([value, label]) => h("div", {}, h("b", {}, value), h("span", {}, label)))
    : t.chips.map((c) => h("span", { class: "chip" }, c)));
  append(clear(t.footLamp), [lamp && h("span", { class: `lamp ${lamp}` })]);
  t.foot.textContent = foot;
  append(clear(t.badge), [count ? h("span", { class: "count" }, String(count)) : null]);
  t.el.classList.toggle("attention", attention);
}

function drawProject(parts, s) {
  const part = s.data.project;
  const p = part?.project;
  if (s.errors.project) {
    clear(parts.head).append(h("span", { class: "label" }, "Project"), h("h1", { id: "hub-project" }, "Not readable"), h("p", { class: "refusal", role: "alert" }, s.errors.project.message));
    clear(parts.side);
    return;
  }
  if (!part?.configured) {
    clear(parts.head).append(h("span", { class: "label" }, "Runs only"), h("h1", { id: "hub-project" }, "No repository"),
      h("p", { class: "meta" }, "The cockpit was started without --repo; it shows the runs of the observed state directories."));
    clear(parts.side);
    return;
  }
  if (!p) {
    clear(parts.head).append(h("span", { class: "label" }, "Project"), h("h1", { id: "hub-project" }, "Observing the repository"),
      h("p", { class: "meta" }, "The structural preparation of the working tree runs; on a large model this takes a while."));
    clear(parts.side).append(h("span", { class: "checkbadge st-none" }, h("span", { class: "lamp st-none" }), "Observing…"));
    return;
  }
  const ok = p.status === "passed";
  append(clear(parts.head), [
    h("span", { class: "label" }, "Project"),
    h("h1", { id: "hub-project" }, p.modelID || "Project"),
    h("p", { class: "meta" },
      p.head ? h("span", {}, "Head ", h("span", { class: "digest", title: p.head }, p.head.slice(0, 12))) : null,
      p.modelDigest ? h("span", {}, "Model ", h("span", { class: "digest", title: p.modelDigest }, shortDigest(p.modelDigest))) : null,
      h("span", {}, `Observed ${when(p.observedAt)}`)),
    p.error ? h("p", { class: "refusal problem" }, p.error) : null]);
  const observe = h("button", { class: "btn small", type: "button", disabled: part.observing }, part.observing ? "Observing…" : "Observe again");
  observe.addEventListener("click", async () => {
    observe.disabled = true;
    observe.textContent = "Observing…";
    try { await api.observeProject(); } catch { /* the hub shows the next state */ }
    // A poll already under way may predate the request; the second read
    // follows it.
    await refreshState();
    await refreshState();
    // A refused request leaves the drawn state unchanged; the button says so.
    if (observe.isConnected && !store.data.project?.observing) {
      observe.disabled = false;
      observe.textContent = "Observe again";
    }
  });
  clear(parts.side).append(
    h("a", { class: `checkbadge ${ok ? "st-done" : "st-fail"}`, href: "#/model", title: "Structural check of the working tree" }, h("span", { class: "lamp" }), ok ? "Check passed" : `Check ${p.status || "not completed"}`),
    observe);
}

// nextStep is the one thing the owner should look at now, most urgent first:
// a proposal to decide, the open session, a run that runs, a check that did
// not pass.
function nextStep(s) {
  const p = s.proposalsToDecide[0];
  if (p) return { accent: "var(--owner)", urgent: true, title: `A proposal waits for your decision`, text: p.title, href: `#/proposal/${encodeURIComponent(p.id)}`, action: "Decide" };
  const session = s.openSession;
  if (session?.state === "waiting") return { title: "Your model session waits for the next prompt", text: `${session.turns.length} of ${session.maxTurns} Agent turns used.`, href: "#/session", action: "Continue" };
  if (session && (session.state === "working" || session.state === "preparing")) return { title: "The Agent of your model session is working", text: session.state === "preparing" ? "Preparing the source and the workspace." : "Its proposals appear in the session.", href: "#/session", action: "Watch" };
  const live = s.liveRuns[0];
  if (live) return { accent: "var(--work)", title: `Run ${live.id} is running`, text: "Follow it live or replay what it recorded so far.", href: `#/run/${encodeURIComponent(live.id)}`, action: "Watch" };
  const project = s.project;
  if (project && project.status !== "passed") return { accent: "var(--fail)", title: "The structural check did not pass", text: project.error || "The model or the registry needs a fix.", href: "#/model", action: "Look" };
  return { calm: true, title: "Nothing waits for you", text: "Browse the model, or change it by prompt in a session." };
}

function drawNext(el, n) {
  el.className = `next${n.urgent ? " urgent" : ""}${n.calm ? " calm" : ""}`;
  el.style.setProperty("--accent", n.accent || "var(--ice)");
  append(clear(el), [
    h("span", { class: "dot", "aria-hidden": "true" }),
    h("div", { class: "text" }, h("strong", {}, n.title), h("span", {}, n.text)),
    n.href ? h("a", { class: `btn${n.urgent ? " act" : ""}`, href: n.href }, n.action) : null]);
}

function drawModel(t, s) {
  const p = s.project;
  const counted = Boolean(p?.modelDigest);
  const session = s.openSession;
  fill(t, {
    stats: counted ? [[String(p.definitions), "Definitions"], [String(p.packages), "Packages"]] : [],
    lamp: session ? sessionClass(session) : p ? (p.status === "passed" ? "st-done" : "st-fail") : "st-none",
    foot: session ? `Session ${session.state}` : p ? (p.status === "passed" ? "Check passed · no open session" : `Check ${p.status || "not completed"}`) : s.data.project?.configured === false ? "No repository" : "Observing the repository",
    attention: session?.state === "waiting",
  });
}

function drawRuns(t, s) {
  if (s.errors.runs) return fill(t, { lamp: "st-fail", foot: "The runs are not readable" });
  const runs = (s.data.runs || []).filter((r) => r.id);
  const live = s.liveRuns.length;
  const latest = runs[0];
  const result = latest?.runOutcome || latest?.outcome;
  fill(t, {
    stats: [[String(runs.length), runs.length === 1 ? "Run recorded" : "Runs recorded"], [String(live), "Running now"]],
    lamp: live ? "st-working" : result ? outcomeClass[result] || "st-none" : "st-none",
    foot: live ? `${live} running now` : latest ? `Latest ${when(latest.started)}: ${result || "no outcome recorded"}` : "No run evidence yet",
  });
}

function drawDecisions(t, s) {
  const waiting = s.proposalsToDecide.length;
  const held = s.data.proposals?.length ?? 0;
  fill(t, {
    stats: [[String(waiting), "Waiting for you"], [String(held), held === 1 ? "Proposal held" : "Proposals held"]],
    lamp: waiting ? "st-owner" : "st-none",
    foot: waiting ? `${waiting} ${waiting === 1 ? "waits" : "wait"} for you` : "Nothing waits for you",
    count: waiting,
    attention: waiting > 0,
  });
}

function drawSettings(t, s) {
  const configured = s.data.project?.configured !== false;
  fill(t, { lamp: configured ? "st-none" : null, foot: configured ? "Read from the committed source" : "Needs a repository" });
}

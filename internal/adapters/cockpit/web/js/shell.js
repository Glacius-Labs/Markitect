// The cockpit's frame: a hub and a few areas the owner enters and leaves. The
// top bar says where the owner is — the hub, or an area with its tabs and the
// item open in it — and carries the project's check and what waits for the
// owner. Escape leads one level up; on the hub, the digits enter the areas.

import { h, clear, append, areaIcon, brandMark } from "./dom.js";
import { store } from "./store.js";
import { setTitle, notifyControl } from "./notify.js";

export const areas = [
  { id: "model", key: "1", href: "#/model", label: "Model", lead: "Browse the canonical model and change it by prompt in a session.",
    tabs: [{ id: "browse", href: "#/model", label: "Browse" }, { id: "session", href: "#/session", label: "Session" }] },
  { id: "runs", key: "2", href: "#/runs", label: "Runs", lead: "Follow a running Apply and replay recorded ones." },
  { id: "decisions", key: "3", href: "#/decisions", label: "Decisions", lead: "Decide on proposals." },
  { id: "settings", key: "4", href: "#/settings", label: "Settings", lead: "Budget, role profiles and team of an Apply. A change becomes a proposal." },
];

let built = null;
let up = null; // where Escape leads from the current place

function build(app) {
  clear(app);
  const where = h("div", { class: "where" });
  const statusbar = h("div", { class: "statusbar", "aria-label": "Project state" });
  const content = h("main", { id: "main", class: "content", tabindex: -1 });
  const topbar = h("header", { class: "topbar" });
  const project = h("a", { class: "projectname", href: "#/projects", hidden: true });
  const shellEl = h("div", { class: "shell" },
    append(topbar, [
      h("a", { class: "brand", href: "#/", title: "Hub" }, brandMark(), h("strong", {}, "Markitect"), h("span", { class: "sub" }, "Cockpit")),
      project, where, statusbar, notifyControl()]),
    content);
  app.append(shellEl);
  // The skip link moves focus to the work surface without changing the place.
  document.querySelector(".skip")?.addEventListener("click", (e) => {
    e.preventDefault();
    content.focus();
  });
  // Pages handle their keys on the document first; what they leave reaches
  // the frame here.
  window.addEventListener("keydown", (e) => {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
    if (e.target.closest?.("input, textarea, select, [contenteditable]")) return;
    if (e.key === "Escape" && up) {
      location.hash = up;
      return;
    }
    const area = !up && areas.find((a) => a.key === e.key);
    if (area) location.hash = area.href;
  });
  built = { topbar, project, where, statusbar, content, tabs: new Map(), place: null, trailEl: null, chips: null };
  store.subscribe(drawState);
  store.start();
  return built;
}

// show prepares the work surface for a place: the hub (no area) or an area
// with its current tab and the trail of the item open in it. Escape leads to
// escape, the hub by default. A page that fills the surface (split views)
// gets no scrolling of its own.
export function show(app, { area = null, tab = null, trail = [], escape = null, fill = false, page = null }) {
  const b = built || build(app);
  const a = areas.find((x) => x.id === area) || null;
  // Outside the areas only a page of its own (the project list) leads up.
  up = a ? escape || "#/" : escape;
  const place = `${area}/${tab}/${page}`;
  const entering = place !== b.place;
  if (entering) {
    b.place = place;
    b.topbar.classList.toggle("in-area", Boolean(a));
    b.tabs.clear();
    clear(b.where);
    if (a) {
      append(b.where, [
        h("a", { class: "btn quiet small back", href: "#/", title: "Back to the hub", "aria-label": "Back to the hub" }, areaIcon("back"), h("span", {}, "Hub")),
        h("span", { class: "divider", "aria-hidden": "true" }),
        h("h1", { class: "areaname" }, h("a", { href: a.href }, emblem(a.id), h("span", { class: "areatext" }, a.label))),
        a.tabs && h("nav", { class: "atabs", "aria-label": `${a.label} views` }, a.tabs.map((t) => {
          const link = h("a", { class: "atab", href: t.href, "aria-current": t.id === tab ? "page" : null }, t.label);
          b.tabs.set(t.id, link);
          return link;
        })),
      ]);
    }
    b.trailEl = h("div", { class: "trail" });
    b.where.append(b.trailEl);
  }
  append(clear(b.trailEl), trail.map((t) => [
    h("span", { "aria-hidden": "true" }, "›"),
    t.href ? h("a", { class: "item", href: t.href }, t.text) : h("span", { class: "item", title: t.text }, t.text)]));
  setTitle(`${trail.at(-1)?.text || a?.label || "Hub"} – Markitect cockpit`);
  // A new place enters with a short fade; reduced motion turns it off.
  b.content.className = `content${fill ? " fill" : ""}${entering ? " enter" : ""}`;
  clear(b.content);
  b.content.scrollTop = 0;
  drawState(store);
  return b.content;
}

// emblem is an area's icon on its accent.
export function emblem(area) {
  return h("span", { class: `emblem t-${area}` }, areaIcon(area));
}

export function refreshState() {
  return store.refresh();
}

export function sessionClass(session) {
  return session.state === "working" || session.state === "preparing" ? "st-working" : session.state === "halted" ? "st-fail" : "st-waiting";
}

// drawState puts the live state on the top bar: what waits for the owner, the
// runs that run now, the check, and the lamp of an open session on its tab.
function drawState(s) {
  if (!built) return;
  const p = s.project;
  const session = s.openSession;
  const live = s.liveRuns.length;
  const waiting = s.proposalsToDecide.length;
  const sessionTab = built.tabs.get("session");
  if (sessionTab) {
    sessionTab.querySelector(".lamp")?.remove();
    if (session) sessionTab.prepend(h("span", { class: `lamp ${sessionClass(session)}`, title: `Session ${session.state}` }));
  }
  const current = s.currentProject;
  // The open project's name leads to the project list. Without one, nothing of
  // a project is shown.
  built.project.hidden = !current;
  if (current) {
    if (built.project.textContent !== current.name) built.project.textContent = current.name;
    built.project.title = `${current.repository}\nSwitch the project or set it up`;
  }
  const chips = [];
  if (waiting) chips.push(h("a", { class: "pill hot", href: "#/decisions" }, h("span", { class: "lamp st-owner" }), h("strong", {}, String(waiting)), waiting === 1 ? " waits for you" : " wait for you"));
  if (live) chips.push(h("a", { class: "pill", href: "#/runs" }, h("span", { class: "lamp st-working" }), h("strong", {}, String(live)), live === 1 ? " run running" : " runs running"));
  const projectPart = s.data.project;
  if (projectPart && !projectPart.configured) {
    chips.push(h("span", { class: "pill" }, "No repository"));
  } else if (!p) {
    chips.push(h("span", { class: "pill" }, h("span", { class: "lamp st-none" }), "Observing"));
  } else {
    const ok = p.status === "passed";
    chips.push(h("a", { class: "pill", href: "#/model", title: p.error || "Structural check of the working tree" }, h("span", { class: `lamp ${ok ? "st-done" : "st-fail"}` }), "Check ", h("strong", {}, ok ? "passed" : p.status || "not completed")));
  }
  // Nothing of a project is shown while none is open: no other part can be
  // read until one is.
  if (s.data.projects && !current) chips.length = 0;
  // The chips are replaced only when they change, so focus on one survives a
  // poll.
  const sig = chips.map((c) => c.outerHTML).join("");
  if (sig !== built.chips) {
    built.chips = sig;
    clear(built.statusbar).append(...chips);
  }
}

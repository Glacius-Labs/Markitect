// The cockpit's frame: the desk rail with the places of work and the lamps
// of their live state, the status bar with the page's place and the project's
// observed state, and the work surface each page draws into.

import { h, clear, append, navIcon, brandMark, shortDigest } from "./dom.js";
import { store } from "./store.js";

const places = [
  { page: "overview", href: "#/", label: "Overview", icon: "overview" },
  { page: "model", href: "#/model", label: "Model", icon: "model" },
  { page: "session", href: "#/session", label: "Session", icon: "session" },
  { page: "proposals", href: "#/proposals", label: "Proposals", icon: "proposals" },
  { page: "runs", href: "#/runs", label: "Runs", icon: "runs" },
  { page: "settings", href: "#/settings", label: "Settings", icon: "settings" },
];

let built = null;

function build(app) {
  clear(app);
  const signals = {};
  const items = places.map((p) => {
    signals[p.page] = h("span", { class: "signal" });
    return h("a", { class: "navitem", href: p.href, dataset: { page: p.page } }, navIcon(p.icon), h("span", {}, p.label), signals[p.page]);
  });
  const footer = h("div", { class: "footer" });
  const crumbs = h("div", { class: "crumbs" });
  const statusbar = h("div", { class: "statusbar", "aria-label": "Project state" });
  const shellEl = h("div", { class: "shell" });
  const menu = h("button", { class: "btn quiet icon menu", type: "button", "aria-label": "Places", "aria-expanded": "false" }, navIcon("menu"));
  menu.addEventListener("click", () => {
    const open = shellEl.classList.toggle("open");
    menu.setAttribute("aria-expanded", String(open));
  });
  const content = h("main", { id: "main", class: "content", tabindex: -1 });
  shellEl.append(
    h("aside", { class: "desk" },
      h("a", { class: "brand", href: "#/" }, brandMark(), h("strong", {}, "Markitect"), h("span", {}, "cockpit")),
      h("nav", { "aria-label": "Places" }, items),
      footer),
    h("header", { class: "topbar" }, menu, crumbs, statusbar),
    content);
  app.append(shellEl);
  // A chosen place, a click beside the drawer or Escape closes the drawer on
  // small windows.
  const closeDrawer = () => {
    shellEl.classList.remove("open");
    menu.setAttribute("aria-expanded", "false");
  };
  shellEl.addEventListener("click", (e) => {
    if (!shellEl.classList.contains("open")) return;
    if (e.target.closest(".navitem, .brand") || !e.target.closest(".desk, .menu")) closeDrawer();
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && shellEl.classList.contains("open")) {
      closeDrawer();
      menu.focus();
    }
  });
  built = { shellEl, items, signals, footer, crumbs, statusbar, content };
  store.subscribe(drawState);
  store.start();
  return built;
}

// show prepares the work surface for a page: the current place, the crumbs
// and whether the page fills the surface (split views) or scrolls.
export function show(app, { page, crumbs, fill = false, title }) {
  const b = built || build(app);
  for (const item of b.items) {
    if (item.dataset.page === page) item.setAttribute("aria-current", "page");
    else item.removeAttribute("aria-current");
  }
  append(clear(b.crumbs), crumbs.map((c, i) => [
    i ? h("span", { class: "sep", "aria-hidden": "true" }, "/") : null,
    i === 0 && crumbs.length > 1 && c.href ? h("h1", {}, h("a", { href: c.href }, c.text)) : h(i === 0 ? "h1" : "span", { class: i ? "tail" : null }, c.text),
  ]));
  document.title = `${title || crumbs.map((c) => c.text).reverse().join(" – ")} – Markitect cockpit`;
  b.content.className = `content${fill ? " fill" : ""}`;
  clear(b.content);
  b.content.scrollTop = 0;
  drawState(store);
  return b.content;
}

export function refreshState() {
  return store.refresh();
}

function lampSignal(cls, text) {
  return [h("span", { class: `lamp ${cls}` }), h("span", { class: "visually-hidden" }, text)];
}

// drawState puts the live state on the desk lamps and the status bar.
function drawState(s) {
  if (!built) return;
  const { signals, statusbar, footer } = built;
  const p = s.project;
  const session = s.openSession;
  const proposals = s.openProposals.length;
  const live = s.liveRuns.length;
  const questions = s.waitingQuestions.length;
  append(clear(signals.overview), [questions ? h("span", { class: "count hot", title: `${questions} owner question${questions === 1 ? "" : "s"} waiting` }, String(questions)) : null]);
  append(clear(signals.session), [session ? lampSignal(session.state === "working" || session.state === "preparing" ? "st-working" : session.state === "halted" ? "st-fail" : "st-waiting", `session ${session.state}`) : null]);
  append(clear(signals.proposals), [proposals ? h("span", { class: "count", title: `${proposals} open proposal${proposals === 1 ? "" : "s"}` }, String(proposals)) : null]);
  append(clear(signals.runs), [live ? lampSignal("st-working", `${live} running`) : null]);
  append(clear(signals.model), [p && p.status !== "passed" ? lampSignal("st-fail", "check not passed") : null]);

  const chips = [];
  const projectPart = s.data.project;
  if (projectPart && !projectPart.configured) {
    chips.push(h("span", { class: "chip" }, "No repository"));
  } else if (!p) {
    chips.push(h("span", { class: "chip" }, h("span", { class: "lamp st-none" }), "Observing the repository"));
  } else {
    const ok = p.status === "passed";
    chips.push(h("a", { class: "chip", href: "#/model", title: p.error || "Structural check of the working tree" }, h("span", { class: `lamp ${ok ? "st-done" : "st-fail"}` }), "Check ", h("strong", {}, ok ? "passed" : p.status || "not completed")));
    if (p.head) chips.push(h("span", { class: "chip optional", title: `Source head ${p.head}` }, "Head ", h("strong", { class: "digest" }, p.head.slice(0, 8))));
  }
  if (session) {
    chips.push(h("a", { class: "chip", href: "#/session" }, h("span", { class: `lamp ${session.state === "working" || session.state === "preparing" ? "st-working" : session.state === "halted" ? "st-fail" : "st-waiting"}` }), "Session ", h("strong", {}, session.state)));
  }
  if (live) chips.push(h("a", { class: "chip", href: "#/runs" }, h("span", { class: "lamp st-working" }), h("strong", {}, String(live)), live === 1 ? " run running" : " runs running"));
  if (questions) chips.push(h("a", { class: "chip", href: "#/" }, h("span", { class: "lamp st-owner" }), h("strong", {}, String(questions)), questions === 1 ? " question waits" : " questions wait"));
  clear(statusbar).append(...chips);

  append(clear(footer), [
    p?.modelID ? h("span", {}, `Model ${p.modelID}`) : null,
    p?.modelDigest ? h("span", { class: "digest", title: p.modelDigest }, shortDigest(p.modelDigest)) : null,
    p?.repository ? h("span", { class: "digest", title: p.repository }, p.repository.split(/[\\/]/).slice(-2).join("/")) : null]);
}

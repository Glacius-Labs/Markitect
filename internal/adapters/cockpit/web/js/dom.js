// Small DOM helpers. Every text from the record goes through text nodes, never
// through markup: Agent-authored content is untrusted.

const SVG = "http://www.w3.org/2000/svg";

export function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  return fill(el, attrs, children);
}

export function s(tag, attrs, ...children) {
  const el = document.createElementNS(SVG, tag);
  return fill(el, attrs, children);
}

function fill(el, attrs, children) {
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === undefined || v === null || v === false) continue;
    if (k === "class") el.setAttribute("class", v);
    else if (k.startsWith("on") && typeof v === "function") el.addEventListener(k.slice(2), v);
    else if (k === "dataset") Object.assign(el.dataset, v);
    else if (k === "style" && typeof v === "object") Object.assign(el.style, v);
    else el.setAttribute(k, v === true ? "" : String(v));
  }
  append(el, children);
  return el;
}

export function append(el, children) {
  for (const c of children.flat(Infinity)) {
    if (c === undefined || c === null || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}

export function clear(el) {
  while (el.firstChild) el.removeChild(el.firstChild);
  return el;
}

export function icon(name) {
  const paths = {
    play: "M4 2.5v11l9-5.5z",
    pause: "M3.5 2.5h3v11h-3zM9.5 2.5h3v11h-3z",
    stepf: "M3 2.5v11l7-5.5zM11 2.5h2v11h-2z",
    stepb: "M13 2.5v11l-7-5.5zM3 2.5h2v11H3z",
    start: "M3 2.5h2v11H3zM14 2.5v11L6 8z",
    end: "M11 2.5h2v11h-2zM2 2.5v11L10 8z",
    close: "M3.5 2.4 8 6.9l4.5-4.5 1.1 1.1L9.1 8l4.5 4.5-1.1 1.1L8 9.1l-4.5 4.5-1.1-1.1L6.9 8 2.4 3.5z",
  };
  return s("svg", { viewBox: "0 0 16 16", "aria-hidden": "true", focusable: "false" }, s("path", { d: paths[name] }));
}

const timeFmt = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
const dateFmt = new Intl.DateTimeFormat(undefined, { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });

export function clock(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  return isNaN(d) ? "" : timeFmt.format(d);
}

export function when(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  return isNaN(d) ? "" : dateFmt.format(d);
}

export function duration(ms) {
  if (!isFinite(ms) || ms < 0) return "";
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s} s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} min ${String(s % 60).padStart(2, "0")} s`;
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, "0")} min`;
}

export function since(fromIso, toIso) {
  if (!fromIso || !toIso) return "";
  return duration(new Date(toIso) - new Date(fromIso));
}

export function shortDigest(d) {
  return d ? d.slice(0, 12) : "";
}

export function label(node) {
  if (node === ".") return "Root";
  if (node === "@orchestrator") return "Orchestrator";
  if (node === "@user") return "Owner";
  return node || "";
}

export function stateWord(state) {
  return { working: "Working", waiting: "Waiting", ready: "Ready", done: "Done" }[state] || "Not started";
}

export function announce(text) {
  const el = document.getElementById("announce");
  if (el) el.textContent = text;
}

// areaIcon draws the outline icons of the cockpit's areas and controls.
export function areaIcon(name) {
  const paths = {
    hub: "M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z",
    model: "M5 4h5v5H5zM14 15h5v5h-5zM5 15h5v5H5zM7.5 9v6M10 17.5h4M10 6.5h6.5V15",
    session: "M4 5h16v11H10l-5 4v-4H4z M8 9.5h8M8 12.5h5",
    decisions: "M4 13l2.5-8h11L20 13v6H4zM4 13h5l1 2.5h4l1-2.5h5",
    runs: "M4 6h9M4 12h16M4 18h12M16 4v4M9 10v4M19 16v4",
    settings: "M4 7h9M17 7h3M4 17h3M11 17h9M15 5v4M9 15v4",
    back: "M14.5 5l-7 7 7 7",
    browse: "M4 5h6v14H4zM12 5h8M12 9h8M12 13h8M12 17h5",
    bell: "M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 2h-15zM10 21h4",
    plus: "M12 5v14M5 12h14",
  };
  return s("svg", { viewBox: "0 0 24 24", "aria-hidden": "true", focusable: "false", class: "ico" }, s("path", { d: paths[name] || "" }));
}

// pagehead is the header every page of its own carries: the title, one short
// sentence and, optionally, the actions on the right and a "How this works"
// disclosure for what the owner should be able to reach but not have to read.
export function pagehead(title, lead, { actions = null, about: more = null } = {}) {
  return h("header", { class: "pagehead" },
    h("div", { class: "headtext" }, h("h2", {}, title), lead ? h("p", { class: "lead" }, lead) : null, more),
    actions ? h("div", { class: "actions" }, actions) : null);
}

// about is a small collapsed disclosure; its body is the longer explanation.
export function about(summary, ...children) {
  return h("details", { class: "about" }, h("summary", {}, summary), h("div", { class: "aboutbody" }, children));
}

// empty is the cockpit's empty state: a short title and one line.
export function empty(title, ...children) {
  return h("div", { class: "void" }, h("strong", {}, title), children.length ? h("span", {}, children) : null);
}

// brandMark is the Markitect mark of the repository's wordmark
// (assets/markitect-wordmark-*.svg): an M of five facets.
export function brandMark() {
  return s("svg", { viewBox: "12 16 104 107", "aria-hidden": "true", focusable: "false", fill: "currentColor" },
    ["M14 18 58 59v20L14 39Z", "m114 18-44 41v20l44-40Z", "m14 47 21 22v43H14Z", "M114 47 93 69v43h21Z", "m64 69 10 10v33l-10 9-10-9V79Z"].map((d) => s("path", { d })));
}

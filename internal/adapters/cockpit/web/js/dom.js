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

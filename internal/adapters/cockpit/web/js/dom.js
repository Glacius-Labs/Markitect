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

// navIcon draws the desk rail's outline icons.
export function navIcon(name) {
  const paths = {
    overview: "M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z",
    model: "M5 4h5v5H5zM14 15h5v5h-5zM5 15h5v5H5zM7.5 9v6M10 17.5h4M10 6.5h6.5V15",
    session: "M4 5h16v11H10l-5 4v-4H4z M8 9.5h8M8 12.5h5",
    proposals: "M4 13l2.5-8h11L20 13v6H4zM4 13h5l1 2.5h4l1-2.5h5",
    runs: "M3 8h18M3 16h18M7 5v14M12 5v14M17 5v14",
    settings: "M4 7h9M17 7h3M4 17h3M11 17h9M15 5v4M9 15v4",
    menu: "M4 7h16M4 12h16M4 17h16",
  };
  return s("svg", { viewBox: "0 0 24 24", "aria-hidden": "true", focusable: "false" }, s("path", { d: paths[name] || "" }));
}

// brandMark is the Glacius Labs mark: four shards around a still center.
export function brandMark() {
  const shard = (d) => s("path", { d });
  return s("svg", { viewBox: "0 0 1700.5129 3315.7236", "aria-hidden": "true", focusable: "false", fill: "currentColor" },
    s("g", { transform: "matrix(21.93579,0,0,21.93579,-7924.0595,-7116.4542)" },
      shard("m 365.8093,385.37175 c 8.8944,-17.34602 17.7888,-34.69203 26.6832,-52.03805 -2.7795,15.7096 -5.559,31.4192 -8.3385,47.1288 -6.1149,1.63642 -12.2298,3.27283 -18.3447,4.90925 z"),
      shard("m 434.1907,385.37175 c -8.8944,-17.34602 -17.7888,-34.69203 -26.6832,-52.03805 2.7795,15.7096 5.559,31.4192 8.3385,47.1288 6.1149,1.63642 12.2298,3.27283 18.3447,4.90925 z"),
      shard("m 365.8093,414.62825 c 8.8944,17.34602 17.7888,34.69203 26.6832,52.03805 -2.7795,-15.7096 -5.559,-31.4192 -8.3385,-47.1288 -6.1149,-1.63642 -12.2298,-3.27283 -18.3447,-4.90925 z"),
      shard("m 434.1907,414.62825 c -8.8944,17.34602 -17.7888,34.69203 -26.6832,52.03805 2.7795,-15.7096 5.559,-31.4192 8.3385,-47.1288 6.1149,-1.63642 12.2298,-3.27283 18.3447,-4.90925 z")));
}

// ticks draws corner registration ticks on a .ticked plate.
export function ticks() {
  return h("span", { class: "ticks", "aria-hidden": "true" }, h("i"), h("i"), h("i"), h("i"));
}

// The track diagram: the Manager tree as tracks, the Orchestrator and the
// owner beside Root, and recorded exchanges as arrows along the edges. An
// arrow marks a recorded send, its movement is illustration only: it does not
// measure transport time and does not mean the message was received or read.

import { h, s, clear, label, stateWord } from "./dom.js";
import { TOKEN_SECONDS } from "./playback.js";

const W = 184, H = 74, HGAP = 26, VGAP = 70, PAD = 28;
const WINDOW = 2.6; // playback seconds an exchange stays visible
const AUTO_COLLAPSE = 24;
const NARROW = 560;

export class Diagram {
  constructor(container, { onSelect }) {
    this.container = container;
    this.onSelect = onSelect;
    this.svg = s("svg", { role: "group", "aria-label": "Organization: Managers, the Orchestrator and the owner, with recorded exchanges" });
    this.edges = s("g");
    this.nodesLayer = s("g");
    this.exLayer = s("g");
    this.svg.append(this.edges, this.nodesLayer, this.exLayer);
    container.append(this.svg);
    this.pos = new Map();
    this.collapsed = new Set();
    this.expanded = new Set();
    this.motion = true;
  }

  update(view, entries, boundary, times, selection) {
    this.view = view;
    this.entries = entries;
    this.boundary = boundary;
    this.times = times;
    this.selection = selection;
    this.narrow = this.container.clientWidth > 0 && this.container.clientWidth < NARROW;
    this.svg.style.display = this.narrow ? "none" : "block";
    if (this.narrow) {
      this.renderList();
      return;
    }
    this.listEl?.remove();
    this.listEl = null;
    this.layout();
    this.drawNodes();
    this.drawExchanges(this.lastT ?? times[boundary - 1] ?? 0, false);
  }

  // renderList is the organization as an indented list with the exchanges of
  // the latest step written out: the static view for narrow windows.
  renderList() {
    this.listEl?.remove();
    const rows = this.view.managers.map((m) => {
      const li = h("li", { class: `st-${m.state || "none"}${this.selection === `manager:${m.path}` ? " selected" : ""}`, tabindex: 0, role: "button", style: { paddingLeft: `${8 + m.depth * 16}px` } },
        h("span", { class: "lamp" }), h("span", { class: "oname" }, label(m.path), h("span", { class: "ostate" }, stateWord(m.state))), h("span", { class: "oact" }, activity(m)));
      const go = () => this.onSelect(`manager:${m.path}`);
      li.addEventListener("click", go);
      li.addEventListener("keydown", (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); go(); } });
      return li;
    });
    const recent = [];
    for (let i = Math.min(this.boundary, this.entries.length) - 1; i >= 0 && recent.length < 3; i--) {
      const e = this.entries[i];
      if (!e.link) continue;
      const li = h("li", {}, h("button", { class: "ref", onclick: () => this.onSelect(e.ref) }, `${label(e.link.from)} to ${label(e.link.to)}`), `: ${e.link.purpose}`);
      recent.push(li);
    }
    this.listEl = h("div", {}, h("ul", { class: "orglist", "aria-label": "Organization" }, rows),
      recent.length ? h("ul", { class: "recent", "aria-label": "Latest recorded exchanges" }, recent) : null);
    this.container.append(this.listEl);
  }

  hidden(path) {
    let p = this.parentOf.get(path);
    while (p !== undefined) {
      if (this.collapsedNow.has(p)) return true;
      p = this.parentOf.get(p);
    }
    return false;
  }

  visibleFor(node) {
    if (node.startsWith("@") || !this.hidden(node)) return node;
    let p = this.parentOf.get(node);
    while (p !== undefined && this.hidden(p)) p = this.parentOf.get(p);
    return p ?? ".";
  }

  layout() {
    const managers = this.view.managers;
    this.parentOf = new Map(managers.map((m) => [m.path, m.parent || undefined]));
    const byPath = new Map(managers.map((m) => [m.path, m]));
    // Auto-collapse deep subtrees of a large organization unless they hold the
    // selection or active work; an explicit expand or collapse wins.
    const busy = (m) => m.state === "working" || (this.selection === `manager:${m.path}`) || (m.children || []).some((c) => byPath.get(c) && busy(byPath.get(c)));
    this.collapsedNow = new Set();
    for (const m of managers) {
      if (!m.children?.length) continue;
      if (this.collapsed.has(m.path)) this.collapsedNow.add(m.path);
      else if (!this.expanded.has(m.path) && managers.length > AUTO_COLLAPSE && m.depth >= 2 && !busy(m)) this.collapsedNow.add(m.path);
    }
    this.pos.clear();
    let slot = 0;
    const place = (m, depth) => {
      const kids = this.collapsedNow.has(m.path) ? [] : (m.children || []).map((c) => byPath.get(c)).filter(Boolean);
      let x;
      if (kids.length === 0) {
        x = slot * (W + HGAP);
        slot++;
      } else {
        const xs = kids.map((k) => place(k, depth + 1));
        x = (xs[0] + xs[xs.length - 1]) / 2;
      }
      this.pos.set(m.path, { x, y: (depth + 1) * (H + VGAP), w: W, h: H });
      return x;
    };
    const root = byPath.get(".");
    if (root) place(root, 0);
    const rootPos = this.pos.get(".");
    const span = Math.max(slot, 1) * (W + HGAP) - HGAP;
    const rx = rootPos ? rootPos.x : 0;
    const top = rootPos ? rootPos.y : H + VGAP;
    this.pos.set("@orchestrator", { x: rx - W - 2 * HGAP - 20, y: top, w: W - 24, h: H - 16 });
    // A Run has no owner channel: the owner is drawn only for the record of an
    // earlier run that holds owner-channel records.
    if (this.view.ownerChannel) this.pos.set("@user", { x: rx + W + 2 * HGAP + 44, y: top, w: W - 24, h: H - 16 });
    let minX = Infinity, maxX = -Infinity, maxY = 0;
    for (const p of this.pos.values()) {
      minX = Math.min(minX, p.x);
      maxX = Math.max(maxX, p.x + p.w);
      maxY = Math.max(maxY, p.y + p.h);
    }
    if (!isFinite(minX)) { minX = 0; maxX = span; }
    const shift = PAD - minX;
    for (const p of this.pos.values()) {
      p.x += shift;
      p.y = p.y - (H + VGAP) + PAD + 8;
    }
    this.width = maxX - minX + 2 * PAD;
    this.height = maxY - (H + VGAP) + 2 * PAD + 30;
    this.svg.setAttribute("viewBox", `0 0 ${this.width} ${this.height}`);
    this.svg.setAttribute("width", this.width);
    this.svg.setAttribute("height", this.height);
    this.svg.style.maxWidth = "100%";
    this.svg.style.height = "auto";
    this.svg.style.margin = "0 auto";
  }

  drawNodes() {
    clear(this.edges);
    clear(this.nodesLayer);
    const view = this.view;
    for (const m of view.managers) {
      if (!m.parent || this.hidden(m.path)) continue;
      const a = this.pos.get(m.parent), b = this.pos.get(m.path);
      if (!a || !b) continue;
      const x1 = a.x + a.w / 2, y1 = a.y + a.h, x2 = b.x + b.w / 2, y2 = b.y;
      const mid = (y1 + y2) / 2;
      this.edges.append(s("path", { class: "track-edge", d: `M${x1} ${y1} V${mid} H${x2} V${y2}` }));
    }
    const root = this.pos.get("."), user = this.pos.get("@user");
    if (root && user) {
      this.edges.append(s("path", { class: "owner-edge", d: `M${root.x + root.w} ${root.y + root.h / 2} H${user.x}` }));
    }
    this.nodesLayer.append(this.systemNode("@orchestrator", "Orchestrator", "Checks, accepts Root, routes"));
    if (this.pos.has("@user")) {
      const ownerOpen = view.questions.filter((q) => q.to === "user" && q.status === "open").length;
      this.nodesLayer.append(this.systemNode("@user", "Owner", ownerOpen ? `${ownerOpen} unanswered question${ownerOpen > 1 ? "s" : ""} recorded` : "Earlier run: owner channel"));
    }
    for (const m of view.managers) {
      if (this.hidden(m.path)) continue;
      this.nodesLayer.append(this.managerNode(m));
    }
  }

  systemNode(id, name, line) {
    const p = this.pos.get(id);
    const g = s("g", { class: "node system", transform: `translate(${p.x},${p.y})`, tabindex: 0, role: "button", "aria-label": `${name}: ${line}` },
      s("rect", { class: "plate", width: p.w, height: p.h, rx: 6 }),
      s("text", { class: "name", x: 12, y: 22 }, name),
      s("text", { class: "line", x: 12, y: 42 }, line));
    if (id === "@user") this.bindSelect(g, "owner");
    else this.bindSelect(g, "orchestrator");
    return g;
  }

  managerNode(m) {
    const p = this.pos.get(m.path);
    const state = m.state || "none";
    const selected = this.selection === `manager:${m.path}`;
    const g = s("g", {
      class: `node st-${state}${selected ? " selected" : ""}`, transform: `translate(${p.x},${p.y})`, tabindex: 0, role: "button",
      "aria-label": `${label(m.path)}: ${stateWord(m.state)}. ${activity(m)}`, "aria-pressed": selected ? "true" : "false",
    });
    g.append(s("rect", { class: "plate", width: p.w, height: p.h, rx: 6 }), s("rect", { class: "bar", width: 6, height: p.h, rx: 2 }));
    g.append(s("text", { class: "name", x: 16, y: 21 }, fit(label(m.path), 21)));
    const lamp = s("circle", { class: "lampc", cx: 21, cy: 38, r: 5 });
    const color = { working: "var(--work)", waiting: "var(--wait)", done: "var(--done)", ready: "transparent" }[state] || "transparent";
    lamp.setAttribute("fill", color);
    lamp.setAttribute("stroke", state === "ready" ? "var(--ink)" : state === "none" ? "var(--ink-3)" : color);
    g.append(lamp, s("text", { class: "line state", x: 32, y: 42 }, fit(stateWord(m.state), 14)));
    g.append(s("text", { class: "line", x: 16, y: 61 }, fit(activity(m), 27)));
    let bx = p.w - 8;
    const badge = (cls, text) => {
      const w = 9 + text.length * 7;
      bx -= w;
      g.append(s("g", { class: `badge ${cls}`, transform: `translate(${bx},8)` }, s("rect", { width: w, height: 17, rx: 3 }), s("text", { x: 5, y: 12.5 }, text)));
      bx -= 4;
    };
    if (m.findings?.length) badge("badge-fail", m.findings.length === 1 ? m.findings[0] : `${m.findings.length} F`);
    if (m.pending) badge("badge-pend", m.pending);
    if (this.collapsedNow.has(m.path)) {
      const n = countBelow(m, this.view.managers);
      g.append(s("text", { class: "line", x: p.w / 2, y: p.h + 16, "text-anchor": "middle" }, `+${n} below, select to expand`));
    }
    g.append(s("title", {}, `${label(m.path)}: ${stateWord(m.state)}${m.wait ? `, waits for ${m.wait.text}` : ""}`));
    this.bindSelect(g, `manager:${m.path}`, () => {
      if (this.collapsedNow.has(m.path)) {
        this.collapsed.delete(m.path);
        this.expanded.add(m.path);
      }
    });
    return g;
  }

  bindSelect(g, ref, before) {
    const go = () => {
      before?.();
      this.onSelect(ref);
    };
    g.addEventListener("click", go);
    g.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        go();
      }
    });
  }

  setMotion(on) {
    this.motion = on;
  }

  // drawExchanges draws the exchanges near playback time t. While playing with
  // motion, a token travels along each path; otherwise the latest exchange
  // step is drawn complete and the ones just before it faded.
  drawExchanges(t, playing) {
    this.lastT = t;
    clear(this.exLayer);
    if (!this.entries || !this.view) return;
    if (this.narrow) {
      if (!playing) this.renderList();
      return;
    }
    const times = this.times;
    const linked = [];
    for (let i = Math.min(this.boundary, this.entries.length) - 1; i >= 0; i--) {
      const e = this.entries[i];
      if (!e.link) continue;
      if (playing && this.motion) {
        if (times[i] < t - WINDOW) break;
      } else if (linked.length && times[i] < times[linked[0].i] - WINDOW) {
        break;
      }
      linked.unshift({ e, i });
      if (linked.length > 14) linked.shift();
    }
    if (!linked.length) return;
    const latest = times[linked[linked.length - 1].i];
    for (const { e, i } of linked) {
      let progress = 1;
      if (playing && this.motion) progress = Math.max(0, Math.min(1, (t - times[i]) / TOKEN_SECONDS));
      const past = !(playing && this.motion) && latest - times[i] > 0.05;
      const fading = playing && this.motion && t - times[i] > TOKEN_SECONDS * 1.6;
      this.exLayer.append(this.arrow(e, progress, past || fading));
    }
  }

  arrow(e, progress, past) {
    const from = this.visibleFor(e.link.from), to = this.visibleFor(e.link.to);
    const a = this.pos.get(from), b = this.pos.get(to);
    const g = s("g", {
      class: `exchange ch-${e.link.channel}${past ? " past" : ""}${this.selection === e.ref ? " selected" : ""}`,
      tabindex: 0, role: "button", "aria-label": `${label(e.link.from)} to ${label(e.link.to)}: ${e.link.purpose}. ${e.text}`,
    });
    if (!a || !b || from === to) return g;
    const d = route(a, b);
    const path = s("path", { d });
    g.append(path);
    this.exLayer.append(g);
    const len = path.getTotalLength();
    if (progress < 1) {
      path.setAttribute("stroke-dasharray", `${len * progress} ${len}`);
    }
    const at = path.getPointAtLength(len * progress);
    const before = path.getPointAtLength(Math.max(0, len * progress - 2));
    const angle = Math.atan2(at.y - before.y, at.x - before.x) * 180 / Math.PI;
    g.append(s("path", { class: "head", d: "M0 0 L-11 -5.5 L-11 5.5 Z", transform: `translate(${at.x},${at.y}) rotate(${angle})` }));
    if (!past) {
      // The chip sits beside its own lane: right of a downward arrow, left of
      // an upward one, above a rightward one and below a leftward one.
      const mid = path.getPointAtLength(len * Math.min(progress, 0.5));
      const text = chipText(e.link.purpose);
      const w = 10 + text.length * 6.6;
      const start = path.getPointAtLength(0), end = path.getPointAtLength(len);
      const dx = end.x - start.x, dy = end.y - start.y;
      let cx = mid.x - w / 2, cy = mid.y - 9;
      if (Math.abs(dy) > Math.abs(dx)) cx = dy > 0 ? mid.x + 6 : mid.x - w - 6;
      else cy = dx > 0 ? mid.y - 22 : mid.y + 4;
      g.append(s("g", { class: "chip", transform: `translate(${cx},${cy})` }, s("rect", { width: w, height: 18, rx: 9 }), s("text", { x: 5, y: 13 }, text)));
    }
    g.append(s("title", {}, `${e.text} (recorded send; delivery is not recorded)`));
    this.bindSelect(g, e.ref);
    return g;
  }
}

// route draws a path between two plates: vertical neighbours get an offset
// lane per direction, so an answer upward never hides a question downward.
function route(a, b) {
  const ax = a.x + a.w / 2, bx = b.x + b.w / 2;
  if (Math.abs(a.y - b.y) < 4) {
    const left = a.x < b.x;
    const x1 = left ? a.x + a.w : a.x, x2 = left ? b.x : b.x + b.w;
    const y = a.y + a.h / 2 + (left ? -8 : 8);
    return `M${x1} ${y} L${x2} ${y}`;
  }
  const down = b.y > a.y;
  const off = down ? 14 : -14;
  const x1 = ax + off, y1 = down ? a.y + a.h : a.y;
  const x2 = bx + off, y2 = down ? b.y : b.y + b.h;
  const my = (y1 + y2) / 2;
  return `M${x1} ${y1} C${x1} ${my}, ${x2} ${my}, ${x2} ${y2}`;
}

function chipText(purpose) {
  return purpose.replace(/^question /, "? ").replace(/^answer /, "↩ ").replace(/^information$/, "info");
}

function fit(text, n) {
  return text.length > n ? text.slice(0, n - 1) + "…" : text;
}

function countBelow(m, managers) {
  const byPath = new Map(managers.map((x) => [x.path, x]));
  let n = 0;
  const walk = (p) => {
    for (const c of byPath.get(p)?.children || []) {
      n++;
      walk(c);
    }
  };
  walk(m.path);
  return n;
}

// activity says in a few words what the record shows the Manager doing.
export function activity(m) {
  if (m.turn) return `${m.turn.role} turn ${m.turn.short}, ${m.turn.readOnly ? "read-only" : "writes"}`;
  if (m.checking) return "submission being checked";
  if (m.helpers?.length) return `${m.helpers.length} helper${m.helpers.length > 1 ? "s" : ""} active`;
  if (m.wait) return `waits: ${waitShort(m.wait)}`;
  if (m.state === "ready") return "ready for a turn";
  if (m.state === "done") return m.result ? `result ${m.result} accepted` : "done";
  if (!m.released && !m.state) return "not released in this Pass";
  return "";
}

function waitShort(w) {
  if (w.outdated) return w.on?.length ? `plan needs ${w.on.map(label).join(", ")}` : "reason not recorded";
  switch (w.kind) {
    case "decision": return `decision on ${w.refs[0].split(":")[1]}`;
    case "children": return w.on?.length ? `children (${w.on.length})` : "children";
    case "answer": return `answer to ${w.refs[0].split(":")[1]}`;
    case "parent-basis": return "parent's basis";
    case "dependency": return `${w.text.startsWith("the accepted result") ? "result" : "basis"} of ${label(w.on[0])}`;
    case "contact": return `reply from ${label(w.on[0])}`;
    case "helpers": return "its helpers";
    case "unnamed": return "release";
  }
  return w.text;
}

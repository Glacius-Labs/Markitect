// One time axis for work, waits, checks, reviews and orchestration. Lanes keep
// parallel branches apart; the record's order, not the clock alone, decides
// what is shown up to the boundary.

import { s, clear, label, clock } from "./dom.js";

const LANE = 26, LABEL = 132, TOP = 26;

export class Timeline {
  constructor(container, { onSelect }) {
    this.container = container;
    this.onSelect = onSelect;
    this.zoom = 1;
  }

  setZoom(z) {
    this.zoom = Math.max(1, Math.min(32, z));
    if (this.view) this.render(this.view, this.entries, this.boundary, this.extent, this.selection);
  }

  render(view, entries, boundary, extent, selection) {
    Object.assign(this, { view, entries, boundary, extent, selection });
    clear(this.container);
    const t0 = Date.parse(extent.start), t1 = Math.max(Date.parse(extent.end), t0 + 1000);
    const lanes = [{ id: "@orchestrator", name: "Orchestrator" }, ...view.managers.map((m) => ({ id: m.path, name: label(m.path), depth: m.depth }))];
    const laneIndex = new Map(lanes.map((l, i) => [l.id, i]));
    const avail = Math.max(320, this.container.clientWidth - LABEL - 16);
    const width = avail * this.zoom;
    const x = (iso) => LABEL + ((Date.parse(iso) - t0) / (t1 - t0)) * width;
    const y = (id) => TOP + (laneIndex.get(id) ?? 0) * LANE;
    const height = TOP + lanes.length * LANE + 8;
    const svg = s("svg", { width: LABEL + width + 16, height, role: "group", "aria-label": "Timeline of Passes, turns, waits and exchanges up to the boundary" });
    const nowIso = view.boundary.at || extent.start;
    const nowX = x(nowIso);

    view.passes.forEach((p, i) => {
      if (!p.started) return;
      const x1 = x(p.started), x2 = p.sealedAt ? x(p.sealedAt) : nowX;
      const band = s("rect", { class: `passband${i % 2 ? " alt" : ""}`, x: x1, y: 0, width: Math.max(1, x2 - x1), height });
      const lbl = s("text", { class: "passlabel", x: x1 + 4, y: 14 }, `${p.id} ${p.purpose}${p.sealed ? "" : " (open)"}`);
      const g = s("g", { class: "mark", tabindex: -1 }, band, lbl, s("title", {}, `Pass ${p.id}: ${p.purpose}${p.verdict ? `, sealed ${p.verdict}` : ""}`));
      g.addEventListener("click", () => this.onSelect(`pass:${p.id}`));
      svg.append(g);
    });

    lanes.forEach((l, i) => {
      const yy = TOP + i * LANE;
      svg.append(s("line", { x1: LABEL, x2: LABEL + width, y1: yy + LANE - 0.5, y2: yy + LANE - 0.5, stroke: "var(--plate-2)" }));
      svg.append(s("text", { class: "lane-label", x: 8 + (l.depth || 0) * 8, y: yy + 17 }, fitLabel(l.name, 16 - (l.depth || 0))));
    });

    for (const sp of view.spans) {
      if (!laneIndex.has(sp.manager)) continue;
      const x1 = x(sp.from), x2 = sp.to ? x(sp.to) : nowX;
      const yy = y(sp.manager);
      const r = s("rect", { class: `span-${sp.state}`, x: x1, y: yy + 7, width: Math.max(0.5, x2 - x1), height: LANE - 14 });
      r.append(s("title", {}, `${label(sp.manager)} ${sp.state}${sp.wait ? `: waits for ${sp.wait}` : ""} (${clock(sp.from)} to ${sp.to ? clock(sp.to) : "the boundary"})`));
      svg.append(r);
    }

    for (const t of view.turns) {
      if (!laneIndex.has(t.manager)) continue;
      const x1 = x(t.started), x2 = t.ended ? x(t.ended) : nowX;
      const helper = t.role !== "manager" && t.role !== "reviewer";
      const cls = `turn${t.readOnly ? " ro" : ""}${t.role === "reviewer" ? " reviewer" : ""}${helper ? " helper" : ""}${t.ended ? "" : " open"}`;
      const r = s("rect", { class: cls, x: x1, y: y(t.manager) + 9, width: Math.max(1.5, x2 - x1), height: LANE - 18, rx: 1.5 });
      r.append(s("title", {}, `${label(t.manager)}: ${t.role} turn ${t.short} (${t.readOnly ? "read-only" : "writes"}), ${clock(t.started)}${t.ended ? ` to ${clock(t.ended)}` : ", still running at the boundary"}`));
      const hit = s("g", { class: "mark" }, r);
      hit.addEventListener("click", () => this.onSelect(`turn:${t.id}`));
      svg.append(hit);
    }

    for (const e of entries) {
      if (e.seq > boundary) break;
      const mk = marker(e);
      if (!mk) continue;
      const lane = mk.lane ?? e.manager ?? "@orchestrator";
      if (!laneIndex.has(lane)) continue;
      const cx = x(e.at), cy = y(lane) + LANE / 2;
      const shape = mk.shape === "tri"
        ? s("path", { class: `mark ${mk.cls}`, d: `M${cx} ${cy - 6} L${cx + 5} ${cy + 4} L${cx - 5} ${cy + 4} Z` })
        : mk.shape === "dia"
          ? s("path", { class: `mark ${mk.cls}`, d: `M${cx} ${cy - 6} L${cx + 5} ${cy} L${cx} ${cy + 6} L${cx - 5} ${cy} Z` })
          : s("circle", { class: `mark ${mk.cls}`, cx, cy, r: 4 });
      shape.append(s("title", {}, `${clock(e.at)} ${e.text}`));
      shape.addEventListener("click", () => this.onSelect(e.ref));
      svg.append(shape);
      if (selection && selection === e.ref) svg.append(s("circle", { class: "sel", cx, cy, r: 8 }));
    }

    svg.append(s("line", { class: "now", x1: nowX, x2: nowX, y1: 0, y2: height }));
    svg.append(s("text", { x: Math.min(nowX + 4, LABEL + width - 60), y: height - 4 }, clock(nowIso)));
    this.container.append(svg);
    if (this.zoom > 1) {
      const left = nowX - this.container.clientWidth * 0.6;
      if (Math.abs(this.container.scrollLeft - left) > this.container.clientWidth * 0.3) this.container.scrollLeft = Math.max(0, left);
    }
  }
}

function marker(e) {
  switch (e.kind) {
    case "proposal.submitted": return { cls: "submit", shape: "tri" };
    case "basis.accepted": case "result.accepted": return { cls: "accept", lane: e.manager };
    case "basis.returned": case "result.returned": case "result.reopened": return { cls: "return", lane: e.manager };
    case "question.asked": return { cls: "question", shape: "dia", lane: e.link?.from };
    case "question.answered": return { cls: "question", lane: e.link?.from === "@user" ? "." : e.link?.from };
    case "question.lapsed": return { cls: "return", shape: "dia" };
    case "message": return { cls: "info", lane: e.link?.from };
    case "finding.recorded": case "finding.closed": case "finding.disputed": return { cls: "finding", shape: "dia" };
    case "problem.raised": return { cls: "return", shape: "dia" };
    case "check": case "gate": case "candidate": return { cls: "check", lane: "@orchestrator" };
    case "outcome": return { cls: "check", shape: "dia", lane: "@orchestrator" };
  }
  return null;
}

function fitLabel(text, n) {
  n = Math.max(6, n);
  return text.length > n ? text.slice(0, n - 1) + "…" : text;
}

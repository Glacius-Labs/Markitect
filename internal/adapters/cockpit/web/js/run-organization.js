import { h, s, append, clear, label } from "./dom.js";
import { OrgChart } from "./orgchart.js";
import { visibleAncestor } from "./org-layout.mjs";
import { runNodes, exchangeFrames, exchangeRoute, pointAlong } from "./run-organization-facts.mjs";

// Adapts the inspector's already-bound view and clock to the shared chart.
// No project observation or current configuration enters a recorded Run.
export class RunOrganization {
  constructor(container, { onSelect }) {
    this.onSelect = onSelect; this.motion = true;
    this.note = h("p", { class: "org-record-note faint" });
    this.recentList = h("ol", { class: "org-recent" });
    this.recent = h("details", { class: "org-recent-details" }, h("summary", {}, "Latest recorded exchanges"), this.recentList);
    this.el = h("div", { class: "run-organization" }, this.note);
    container.append(this.el);
    this.chart = new OrgChart(this.el, { name: "Recorded Run organization", onLayout: () => { if (this.view) this.drawExchanges(this.lastT ?? 0, this.playing || false); }, onSelect: (id) => {
      if (id === "@user") { if (this.view?.ownerChannel) onSelect("owner"); }
      else onSelect(id === "@orchestrator" ? "orchestrator" : `manager:${id}`);
    } });
    this.el.append(this.recent);
    this.reduced = matchMedia("(prefers-reduced-motion: reduce)");
  }

  update(view, entries, boundary, times, selection) {
    if (this.view?.boundary.seq !== view.boundary.seq && !this.playing) this.animateProgress = false;
    this.view = view; this.entries = entries; this.boundary = boundary; this.times = times; this.selection = selection;
    const bound = view.run.binding;
    const note = bound ? `Organization from the recorded run.start binding${bound.head ? ` at ${bound.head.slice(0, 12)}` : ""}. Provider models and cost come only from reported Turns.` : "No run.start binding recorded: showing observed Managers only. Profiles, helper limits and declared checks are unknown.";
    if (this.note.textContent !== note) this.note.textContent = note;
    const selected = selection?.startsWith("manager:") ? selection.slice(8) : selection === "orchestrator" ? "@orchestrator" : selection === "owner" ? "@user" : null;
    this.nodes = runNodes(view);
    this.chart.update(this.nodes, { key: view.run.id, selection: selected, ownerLine: false });
    this.drawExchanges(this.lastT ?? times[Math.min(boundary, view.boundary.seq) - 1] ?? 0, this.playing || false);
  }

  setMotion(on) { this.motion = on; }

  drawExchanges(t, playing) {
    this.lastT = t; this.playing = playing;
    if (playing) this.animateProgress = true;
    clear(this.chart.exchanges);
    if (!this.view) return;
    const frames = exchangeFrames(this.entries, Math.min(this.boundary, this.view.boundary.seq), t, this.times, { animate: Boolean(this.animateProgress && this.motion && !this.reduced.matches), ownerChannel: this.view.ownerChannel });
    const signature = frames.map((f) => f.entry.seq).join(",");
    if (signature !== this.recentSignature) {
      this.recentSignature = signature;
      append(clear(this.recentList), frames.map(({ entry }) => h("li", {}, h("button", { class: "ref", type: "button", onclick: () => this.onSelect(entry.ref) }, `${label(entry.link.from)} to ${label(entry.link.to)}: ${entry.link.purpose}`), " · recorded send; receipt and reading unobserved")));
      this.recent.hidden = !frames.length;
    }
    if (this.chart.narrow) return;
    const visible = new Set(this.chart.positions.keys());
    for (const { entry, progress, past } of frames) {
      const from = visibleAncestor(entry.link.from, this.nodes, visible), to = visibleAncestor(entry.link.to, this.nodes, visible);
      if (!from || !to || from === to) continue;
      const points = exchangeRoute(this.chart.positions.get(from), this.chart.positions.get(to), entry.link.channel);
      const path = points.map((p, i) => `${i ? "L" : "M"}${p.x} ${p.y}`).join(" ");
      const at = pointAlong(points, progress);
      const g = s("g", { class: `org-exchange org-ch-${entry.link.channel}${past ? " past" : ""}${this.selection === entry.ref ? " selected" : ""}`, onclick: () => this.onSelect(entry.ref) },
        s("path", { d: path }),
        s("path", { class: "org-arrowhead", d: "M0 0 L-10 -5 L-10 5 Z", transform: `translate(${points.at(-1).x},${points.at(-1).y}) rotate(${pointAlong(points, 1).angle})` }),
        s("circle", { cx: at.x, cy: at.y, r: 4 }),
        s("title", {}, `${entry.text} · recorded send; delivery and reading unobserved`));
      this.chart.exchanges.append(g);
    }
  }

  destroy() { this.chart.destroy(); this.el.remove(); }
}

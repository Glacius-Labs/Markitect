import { h, s, append, clear } from "./dom.js";
import { CARD, layoutTree, initialCollapse, connector } from "./org-layout.js";

let chartSerial = 0;

// One presentation component for observed project and recorded Run facts.
// It consumes explicit nodes; it does not infer work, authority or hierarchy.
export class OrgChart {
  constructor(container, { onSelect = () => {}, name = "Project organization" } = {}) {
    this.onSelect = onSelect; this.cards = new Map(); this.nodes = [];
    this.notePrefix = `org-review-${++chartSerial}`;
    this.scale = 1; this.pan = { x: 0, y: 0 }; this.collapsed = new Set();
    this.tree = h("div", { class: "org-tree", role: "tree", "aria-label": name });
    this.lines = s("svg", { class: "org-lines", "aria-hidden": "true", focusable: "false" });
    this.exchanges = s("g", { class: "org-exchanges" });
    this.canvas = h("div", { class: "org-canvas" }, this.lines, this.tree);
    this.viewport = h("div", { class: "org-viewport" }, this.canvas);
    this.zoomLabel = h("output", { "aria-label": "Organization zoom" }, "100%");
    this.toolbar = h("div", { class: "org-tools", role: "group", "aria-label": "Organization controls" },
      this.button("Zoom out", "−", () => this.zoom(1 / 1.2)), this.zoomLabel,
      this.button("Zoom in", "+", () => this.zoom(1.2)), this.button("Fit organization", "Fit", () => this.fit()),
      this.button("Expand all departments", "Expand all", () => { this.collapsed.clear(); this.render(); this.fit(); }),
      this.button("Collapse departments", "Collapse", () => { this.collapsed = initialCollapse(this.nodes); this.render(); this.fit(); }));
    this.el = h("div", { class: "org-chart" }, this.toolbar, this.viewport,
      h("p", { class: "org-hint faint" }, "Select a card for details. Arrow keys move through the tree; left and right collapse or expand. Drag the background to pan."));
    append(container, [this.el]);
    this.tree.addEventListener("keydown", (e) => this.key(e));
    this.viewport.addEventListener("pointerdown", (e) => {
      if (this.narrow || e.button !== 0 || e.target.closest(".org-node, .org-reviewer, .org-exchange")) return;
      this.drag = { x: e.clientX, y: e.clientY, pan: { ...this.pan } }; this.viewport.setPointerCapture(e.pointerId);
    });
    this.viewport.addEventListener("pointermove", (e) => {
      if (!this.drag) return;
      this.pan = { x: this.drag.pan.x + e.clientX - this.drag.x, y: this.drag.pan.y + e.clientY - this.drag.y }; this.transform();
    });
    const release = () => { this.drag = null; };
    this.viewport.addEventListener("pointerup", release); this.viewport.addEventListener("pointercancel", release);
    this.viewport.addEventListener("wheel", (e) => { if (e.ctrlKey && !this.narrow) { e.preventDefault(); this.zoom(e.deltaY < 0 ? 1.1 : 1 / 1.1); } }, { passive: false });
    this.resize = new ResizeObserver(() => {
      const narrow = this.viewport.clientWidth < 620;
      if (narrow !== this.narrow) { this.narrow = narrow; this.render(); this.fit(); }
    });
    this.resize.observe(this.viewport);
  }

  button(name, text, action) { return h("button", { type: "button", class: "btn small", "aria-label": name, title: name, onclick: action }, text); }

  update(nodes, { key = "organization", selection = null, ownerLine = true } = {}) {
    if (key !== this.identity) { this.identity = key; this.collapsed = initialCollapse(nodes); this.focusID = nodes[0]?.id; this.fitted = false; }
    this.nodes = nodes; this.selection = selection; this.ownerLine = ownerLine;
    this.narrow = this.viewport.clientWidth < 620;
    this.render();
    if (!this.fitted) { this.fit(); this.fitted = true; }
  }

  render() {
    const focused = document.activeElement?.closest?.(".org-node");
    const focusedID = focused?.dataset.id, wasToggle = document.activeElement?.classList.contains("org-toggle");
    this.layout = layoutTree(this.nodes, this.collapsed);
    this.positions = new Map(this.layout.positions.map((p) => [p.id, p]));
    if (!this.positions.has(this.focusID)) this.focusID = this.layout.positions[0]?.id;
    this.el.classList.toggle("org-narrow", Boolean(this.narrow));
    this.lines.setAttribute("width", this.layout.width); this.lines.setAttribute("height", this.layout.height);
    clear(this.lines);
    for (const edge of this.layout.edges) {
      if (!this.ownerLine && edge.from === "@user") continue;
      this.lines.append(s("path", { d: connector(this.positions.get(edge.from), this.positions.get(edge.to)), class: "org-connection" }));
    }
    this.lines.append(this.exchanges);
    const visible = new Set();
    for (const p of this.layout.positions) {
      const n = this.nodes.find((n) => n.id === p.id); visible.add(n.id);
      let card = this.cards.get(n.id);
      if (!card) {
        card = h("div", { class: "org-node", role: "treeitem", dataset: { id: n.id } });
        card.addEventListener("focus", () => { this.focusID = n.id; this.rove(); });
        card.addEventListener("click", (e) => { if (!e.target.closest("button")) { card.focus(); this.onSelect(n.id); } });
        this.cards.set(n.id, card);
      }
      card.className = `org-node org-${n.kind || "manager"} st-${n.state || "none"}${this.selection === n.id ? " selected" : ""}`;
      card.setAttribute("aria-selected", this.selection === n.id);
      for (const [k, v] of Object.entries({ "aria-label": `${n.title}: ${n.name}${n.state ? `, ${n.state}` : ""}`, "aria-level": p.level + 1, "aria-posinset": p.index, "aria-setsize": p.size })) card.setAttribute(k, v);
      if (p.children.length) card.setAttribute("aria-expanded", !p.collapsed); else card.removeAttribute("aria-expanded");
      const sig = JSON.stringify([n, p.collapsed]);
      if (card._signature !== sig) {
        card._signature = sig;
        append(clear(card), [h("div", { class: "org-person" }, h("span", { class: "org-avatar", "aria-hidden": "true" }, n.initial || n.name.slice(0, 1)), h("div", {}, h("span", { class: "org-role" }, n.title), h("strong", { class: "org-name", title: n.name }, n.name)), h("span", { class: "lamp", title: n.state || "Observed organization" })),
          h("div", { class: "org-facts" }, (n.facts || []).map((text) => h("span", { title: text }, text))),
          n.badges?.length ? h("div", { class: "org-badges" }, n.badges.map((b) => h("span", { class: "chip", title: b }, b))) : null,
          p.children.length ? h("button", { class: "org-toggle", type: "button", tabindex: -1, "aria-label": `${p.collapsed ? "Expand" : "Collapse"} ${n.name}`, onclick: (e) => { e.stopPropagation(); this.toggle(n.id); } }, `${p.collapsed ? "+" : "−"} ${p.children.length} ${p.children.length === 1 ? "department" : "departments"}`) : null]);
      }
      if (!card.isConnected) this.tree.append(card);
      card.style.left = this.narrow ? "" : `${p.x}px`; card.style.top = this.narrow ? "" : `${p.y}px`;
      card.style.marginLeft = this.narrow ? `${Math.min(p.level, 5) * 12}px` : "";
      // In narrow mode order follows the same deterministic pre-order as layout.
      if (this.narrow) this.tree.append(card);
      if (n.reviewer) {
        const noteID = `${this.notePrefix}-${encodeURIComponent(n.id)}`;
        const reviewer = h("aside", { id: noteID, class: `org-reviewer st-${n.reviewer.state || "none"}`, role: "note", "aria-label": `Independent reviewer of ${n.name}` }, h("span", { class: "lamp" }), h("strong", {}, "Independent reviewer"), h("span", {}, n.reviewer.text || "Separate quality seat"));
        if (!this.narrow) { reviewer.style.left = `${CARD.width + CARD.sideGap}px`; reviewer.style.top = "28px"; }
        // The note is descriptive content of its tree item, not an unowned
        // child of role=tree or an additional subordinate Manager tree item.
        card.setAttribute("aria-describedby", noteID);
        card._reviewer?.remove(); card._reviewer = reviewer; card.append(reviewer);
        this.lines.append(s("path", { class: "org-review-line", d: `M${p.x + CARD.width} ${p.y + 65} H${p.x + CARD.width + CARD.sideGap}` }));
      } else { card._reviewer?.remove(); card._reviewer = null; card.removeAttribute("aria-describedby"); }
    }
    for (const [id, card] of this.cards) if (!visible.has(id)) { card.remove(); card._reviewer?.remove(); }
    this.canvas.style.width = this.narrow ? "100%" : `${this.layout.width}px`; this.canvas.style.height = this.narrow ? "auto" : `${this.layout.height}px`;
    this.rove(); this.transform();
    if (focusedID && this.positions.has(focusedID)) {
      const card = this.cards.get(focusedID); (wasToggle ? card.querySelector(".org-toggle") || card : card).focus({ preventScroll: true });
    }
  }

  rove() { for (const [id, card] of this.cards) card.tabIndex = id === this.focusID && this.positions?.has(id) ? 0 : -1; }
  toggle(id) { this.focusID = id; if (this.collapsed.has(id)) this.collapsed.delete(id); else this.collapsed.add(id); this.render(); }
  key(e) {
    const card = e.target.closest(".org-node"); if (!card || e.target.closest("button")) return;
    const id = card.dataset.id, p = this.positions.get(id), list = this.layout.positions, at = list.findIndex((n) => n.id === id);
    let next;
    if (e.key === "ArrowDown") next = list[Math.min(at + 1, list.length - 1)]?.id;
    else if (e.key === "ArrowUp") next = list[Math.max(0, at - 1)]?.id;
    else if (e.key === "Home") next = list[0]?.id;
    else if (e.key === "End") next = list.at(-1)?.id;
    else if (e.key === "ArrowRight") { if (p.collapsed) this.toggle(id); else next = p.children[0]; }
    else if (e.key === "ArrowLeft") { if (p.children.length && !p.collapsed) this.toggle(id); else next = this.nodes.find((n) => n.id === id)?.parent; }
    else if (e.key === "Enter" || e.key === " ") this.onSelect(id);
    else return;
    e.preventDefault(); e.stopPropagation();
    if (next && this.positions.has(next)) { this.focusID = next; this.rove(); this.cards.get(next)?.focus(); }
  }
  transform() {
    this.canvas.style.transform = this.narrow ? "none" : `translate(${this.pan.x}px, ${this.pan.y}px) scale(${this.scale})`;
    this.zoomLabel.value = `${Math.round(this.scale * 100)}%`;
  }
  fit() {
    if (!this.layout || this.narrow) return;
    this.scale = Math.min(1, Math.max(1, this.viewport.clientWidth - 12) / this.layout.width, Math.max(1, this.viewport.clientHeight - 12) / this.layout.height);
    this.fitScale = this.scale;
    this.pan = { x: (this.viewport.clientWidth - this.layout.width * this.scale) / 2, y: 6 }; this.transform();
  }
  zoom(factor) {
    if (this.narrow) return;
    const old = this.scale, next = Math.max(Math.min(0.08, this.fitScale || 0.08), Math.min(2.5, old * factor));
    const cx = this.viewport.clientWidth / 2, cy = this.viewport.clientHeight / 2;
    this.pan = { x: cx - (cx - this.pan.x) * next / old, y: cy - (cy - this.pan.y) * next / old }; this.scale = next; this.transform();
  }
  destroy() { this.resize.disconnect(); this.el.remove(); }
}

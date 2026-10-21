// Pure layout over explicit parent identities. No model or ownership inference.
export const CARD = { width: 244, height: 164, reviewer: 132, sideGap: 18, gap: 28, rowGap: 58, pad: 24 };

export function initialCollapse(nodes) {
  return new Set(nodes.length > 14 ? nodes.filter((n) => n.kind === "manager" && n.managerDepth >= 1 && nodes.some((c) => c.parent === n.id)).map((n) => n.id) : []);
}

export function layoutTree(nodes, collapsed = new Set()) {
  const byID = new Map(), children = new Map();
  for (const n of nodes) {
    if (!n.id || byID.has(n.id)) throw new Error("Organization requires unique card identities");
    byID.set(n.id, n); children.set(n.id, []);
  }
  const roots = [];
  for (const n of nodes) {
    if (n.parent == null) roots.push(n.id);
    else {
      if (!byID.has(n.parent)) throw new Error(`Missing organization parent: ${n.parent}`);
      children.get(n.parent).push(n.id);
    }
  }
  const visiting = new Set(), visited = new Set();
  const validate = (id) => {
    if (visiting.has(id)) throw new Error("Organization contains a cycle");
    if (visited.has(id)) return;
    visiting.add(id);
    for (const c of children.get(id)) validate(c);
    visiting.delete(id); visited.add(id);
  };
  for (const n of nodes) validate(n.id);
  const widths = new Map();
  const visibleChildren = (id) => collapsed.has(id) ? [] : children.get(id);
  const measure = (id) => {
    const n = byID.get(id), kids = visibleChildren(id);
    const own = CARD.width + (n.reviewer ? CARD.sideGap + CARD.reviewer : 0);
    const below = kids.reduce((sum, c) => sum + measure(c), 0) + Math.max(0, kids.length - 1) * CARD.gap;
    const width = Math.max(own, below); widths.set(id, width); return width;
  };
  roots.forEach(measure);
  const positions = [], edges = [];
  let height = 0;
  const place = (id, left, level, index, size) => {
    const n = byID.get(id), kids = visibleChildren(id), width = widths.get(id);
    const own = CARD.width + (n.reviewer ? CARD.sideGap + CARD.reviewer : 0);
    const p = { id, x: left + (width - own) / 2, y: CARD.pad + level * (CARD.height + CARD.rowGap), level, index, size, children: children.get(id), collapsed: collapsed.has(id), w: CARD.width, h: CARD.height };
    positions.push(p); height = Math.max(height, p.y + p.h + CARD.pad);
    const below = kids.reduce((sum, c) => sum + widths.get(c), 0) + Math.max(0, kids.length - 1) * CARD.gap;
    let at = left + (width - below) / 2;
    kids.forEach((c, i) => { place(c, at, level + 1, i + 1, kids.length); edges.push({ from: id, to: c }); at += widths.get(c) + CARD.gap; });
  };
  let at = CARD.pad;
  roots.forEach((id, i) => { place(id, at, 0, i + 1, roots.length); at += widths.get(id) + CARD.gap; });
  return { positions, edges, width: Math.max(1, at - CARD.gap + CARD.pad), height: Math.max(1, height) };
}

export function connector(a, b) {
  const x1 = a.x + a.w / 2, y1 = a.y + a.h, x2 = b.x + b.w / 2, y2 = b.y;
  const mid = (y1 + y2) / 2;
  return `M${x1} ${y1} V${mid} H${x2} V${y2}`;
}

export function visibleAncestor(id, nodes, visible) {
  const byID = new Map(nodes.map((n) => [n.id, n]));
  const seen = new Set();
  while (id && !visible.has(id) && !seen.has(id)) { seen.add(id); id = byID.get(id)?.parent; }
  return visible.has(id) ? id : null;
}

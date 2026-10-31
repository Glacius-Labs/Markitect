// The Map's layout: a layered drawing of the Package graph. A Package stands
// above the Packages it depends on, so dependencies point down towards the
// stable ones. A cycle is broken for the layering only; its edges are still
// drawn, pointing up.

export interface Placed {
  id: string;
  label: string;
  layer: number;
  x: number; // left edge
  y: number; // top edge
  w: number;
  h: number;
}

export interface Routed {
  from: string;
  to: string;
  d: string; // the SVG path
  up: boolean; // the edge points against the layering, as a cycle's does
}

export interface Drawing {
  nodes: Placed[];
  edges: Routed[];
  width: number;
  height: number;
}

const nodeHeight = 34;
const layerGap = 72;
const nodeGap = 24;
const margin = 16;

// width estimates a label's box; the Map draws labels in the sans font at
// 13 pixels.
function width(label: string): number {
  return Math.max(72, Math.round(label.length * 7.4) + 28);
}

export function layout(nodes: { id: string; label: string }[], edges: { from: string; to: string }[]): Drawing {
  const ids = nodes.map((n) => n.id).sort();
  const known = new Set(ids);
  const out = new Map<string, string[]>();
  for (const id of ids) {
    out.set(id, []);
  }
  const pairs = new Set<string>();
  for (const e of edges) {
    const key = `${e.from}\u0000${e.to}`;
    if (e.from !== e.to && known.has(e.from) && known.has(e.to) && !pairs.has(key)) {
      pairs.add(key);
      out.get(e.from)!.push(e.to);
    }
  }
  for (const list of out.values()) {
    list.sort();
  }

  // Back edges, found depth first in a stable order, are reversed for the
  // layering.
  const back = new Set<string>();
  const state = new Map<string, 'open' | 'done'>();
  const visit = (v: string) => {
    state.set(v, 'open');
    for (const w of out.get(v)!) {
      const s = state.get(w);
      if (s === 'open') {
        back.add(`${v}\u0000${w}`);
      } else if (s === undefined) {
        visit(w);
      }
    }
    state.set(v, 'done');
  };
  for (const id of ids) {
    if (!state.has(id)) {
      visit(id);
    }
  }
  const dag = new Map<string, string[]>(ids.map((id) => [id, []]));
  for (const [from, list] of out) {
    for (const to of list) {
      if (back.has(`${from}\u0000${to}`)) {
        dag.get(to)!.push(from);
      } else {
        dag.get(from)!.push(to);
      }
    }
  }

  // The longest path from a Package nothing depends on gives each layer.
  const indegree = new Map<string, number>(ids.map((id) => [id, 0]));
  for (const list of dag.values()) {
    for (const to of list) {
      indegree.set(to, indegree.get(to)! + 1);
    }
  }
  const layer = new Map<string, number>(ids.map((id) => [id, 0]));
  const queue = ids.filter((id) => indegree.get(id) === 0);
  while (queue.length > 0) {
    const v = queue.shift()!;
    for (const w of dag.get(v)!) {
      layer.set(w, Math.max(layer.get(w)!, layer.get(v)! + 1));
      indegree.set(w, indegree.get(w)! - 1);
      if (indegree.get(w) === 0) {
        queue.push(w);
      }
    }
  }
  const depth = Math.max(0, ...layer.values());
  const layers: string[][] = Array.from({ length: depth + 1 }, () => []);
  for (const id of ids) {
    layers[layer.get(id)!].push(id);
  }

  // A few sweeps order each layer by the mean position of its neighbours.
  const neighbours = new Map<string, string[]>(ids.map((id) => [id, []]));
  for (const [from, list] of dag) {
    for (const to of list) {
      neighbours.get(from)!.push(to);
      neighbours.get(to)!.push(from);
    }
  }
  const position = new Map<string, number>();
  const place = () => layers.forEach((l) => l.forEach((id, i) => position.set(id, i)));
  place();
  for (let sweep = 0; sweep < 6; sweep++) {
    const order = sweep % 2 === 0 ? layers : [...layers].reverse();
    for (const l of order) {
      const centre = (id: string) => {
        const ns = neighbours.get(id)!;
        return ns.length === 0 ? position.get(id)! : ns.reduce((s, n) => s + position.get(n)!, 0) / ns.length;
      };
      const keyed = l.map((id) => ({ id, c: centre(id) }));
      keyed.sort((a, b) => a.c - b.c || (a.id < b.id ? -1 : 1));
      l.splice(0, l.length, ...keyed.map((k) => k.id));
      l.forEach((id, i) => position.set(id, i));
    }
  }

  const labels = new Map(nodes.map((n) => [n.id, n.label]));
  const rowWidth = (l: string[]) => l.reduce((s, id) => s + width(labels.get(id)!), 0) + nodeGap * Math.max(0, l.length - 1);
  const widest = Math.max(0, ...layers.map(rowWidth));
  const placed = new Map<string, Placed>();
  layers.forEach((l, i) => {
    let x = margin + (widest - rowWidth(l)) / 2;
    for (const id of l) {
      const w = width(labels.get(id)!);
      placed.set(id, { id, label: labels.get(id)!, layer: i, x, y: margin + i * (nodeHeight + layerGap), w, h: nodeHeight });
      x += w + nodeGap;
    }
  });

  const routed: Routed[] = [];
  for (const [from, list] of out) {
    for (const to of list) {
      const a = placed.get(from)!;
      const b = placed.get(to)!;
      const ax = a.x + a.w / 2;
      const bx = b.x + b.w / 2;
      let d: string;
      if (b.layer > a.layer) {
        const y1 = a.y + a.h;
        const y2 = b.y;
        const dy = Math.max(24, (y2 - y1) / 2);
        d = `M${ax},${y1} C${ax},${y1 + dy} ${bx},${y2 - dy} ${bx},${y2}`;
      } else if (b.layer < a.layer) {
        const y1 = a.y;
        const y2 = b.y + b.h;
        const dy = Math.max(24, (y1 - y2) / 2);
        d = `M${ax},${y1} C${ax},${y1 - dy} ${bx},${y2 + dy} ${bx},${y2}`;
      } else {
        const y = a.y + a.h;
        d = `M${ax},${y} C${ax},${y + 36} ${bx},${y + 36} ${bx},${y}`;
      }
      routed.push({ from, to, d, up: b.layer <= a.layer });
    }
  }
  return {
    nodes: ids.map((id) => placed.get(id)!),
    edges: routed,
    width: widest + 2 * margin,
    height: margin * 2 + (depth + 1) * nodeHeight + depth * layerGap + 40,
  };
}

import { describe, expect, it } from 'vitest';
import { layout } from './layout';

const nodes = (...ids: string[]) => ids.map((id) => ({ id, label: id }));

describe('the Map layout', () => {
  it('puts a Package above the Packages it depends on', () => {
    const d = layout(nodes('a', 'b', 'c'), [
      { from: 'a', to: 'b' },
      { from: 'b', to: 'c' },
      { from: 'a', to: 'c' },
    ]);
    const layer = Object.fromEntries(d.nodes.map((n) => [n.id, n.layer]));
    expect(layer).toEqual({ a: 0, b: 1, c: 2 });
    expect(d.edges).toHaveLength(3);
    expect(d.edges.every((e) => !e.up)).toBe(true);
  });

  it('draws both edges of a cycle, one of them pointing up', () => {
    const d = layout(nodes('loans', 'members'), [
      { from: 'loans', to: 'members' },
      { from: 'members', to: 'loans' },
    ]);
    expect(d.nodes.map((n) => n.layer).sort()).toEqual([0, 1]);
    expect(d.edges.filter((e) => e.up)).toHaveLength(1);
  });

  it('keeps nodes of one layer apart and inside the drawing', () => {
    const d = layout(nodes('', 'a', 'b', 'c'), [
      { from: 'a', to: 'c' },
      { from: 'b', to: 'c' },
    ]);
    const top = d.nodes.filter((n) => n.layer === 0).sort((x, y) => x.x - y.x);
    expect(top).toHaveLength(3);
    for (let i = 1; i < top.length; i++) {
      expect(top[i].x).toBeGreaterThanOrEqual(top[i - 1].x + top[i - 1].w);
    }
    for (const n of d.nodes) {
      expect(n.x).toBeGreaterThanOrEqual(0);
      expect(n.x + n.w).toBeLessThanOrEqual(d.width);
      expect(n.y + n.h).toBeLessThanOrEqual(d.height);
    }
  });

  it('ignores edges to unknown Packages and repeated edges', () => {
    const d = layout(nodes('a', 'b'), [
      { from: 'a', to: 'b' },
      { from: 'a', to: 'b' },
      { from: 'a', to: 'gone' },
    ]);
    expect(d.edges).toHaveLength(1);
  });
});

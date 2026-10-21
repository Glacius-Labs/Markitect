import test from "node:test";
import assert from "node:assert/strict";
import { CARD, layoutTree, initialCollapse, connector, visibleAncestor } from "./web/js/org-layout.mjs";

const nodes = [
  { id: "owner", parent: null, kind: "owner" }, { id: "ops", parent: "owner", kind: "orchestrator" },
  { id: ".", parent: "ops", kind: "manager", managerDepth: 0, reviewer: true },
  { id: "a", parent: ".", kind: "manager", managerDepth: 1, reviewer: true },
  { id: "b", parent: ".", kind: "manager", managerDepth: 1, reviewer: true },
  { id: "a/deep", parent: "a", kind: "manager", managerDepth: 2, reviewer: true },
];

test("top-down layout is deterministic, orthogonal, and reserves reviewer seats", () => {
  const original = structuredClone(nodes), a = layoutTree(nodes), b = layoutTree(nodes);
  assert.deepEqual(a, b); assert.deepEqual(nodes, original);
  for (const edge of a.edges) {
    const from = a.positions.find((p) => p.id === edge.from), to = a.positions.find((p) => p.id === edge.to);
    assert.ok(from.y + from.h < to.y); assert.match(connector(from, to), /^M[\d.]+ [\d.]+ V[\d.]+ H[\d.]+ V[\d.]+$/);
  }
  for (const level of new Set(a.positions.map((p) => p.level))) {
    const row = a.positions.filter((p) => p.level === level).sort((x, y) => x.x - y.x);
    for (let i = 1; i < row.length; i++) assert.ok(row[i].x >= row[i-1].x + CARD.width + CARD.reviewer + CARD.sideGap);
  }
});

test("collapse keeps explicit reporting ancestry for hidden exchanges", () => {
  const v = layoutTree(nodes, new Set(["a"])), ids = new Set(v.positions.map((p) => p.id));
  assert.ok(!ids.has("a/deep")); assert.equal(visibleAncestor("a/deep", nodes, ids), "a");
  assert.equal(visibleAncestor("unrecorded", nodes, ids), null);
  assert.deepEqual(v.positions.map((p) => p.id), ["owner", "ops", ".", "a", "b"]);
});

test("large trees initially show Root and the first department level", () => {
  assert.deepEqual(initialCollapse(nodes), new Set());
  const large = [...nodes, ...Array.from({ length: 15 }, (_, i) => ({ id: `a/p${i}`, parent: "a", kind: "manager", managerDepth: 2 }))];
  assert.deepEqual(initialCollapse(large), new Set(["a"]));
  assert.equal(layoutTree(large, initialCollapse(large)).positions.length, 5);
});

test("invalid topology refuses instead of guessing a parent or looping", () => {
  assert.throws(() => layoutTree([{ id: "a", parent: "missing" }]), /Missing/);
  assert.throws(() => layoutTree([{ id: "a", parent: "b" }, { id: "b", parent: "a" }]), /cycle/);
  assert.throws(() => layoutTree([{ id: "a" }, { id: "a" }]), /unique/);
  assert.deepEqual(layoutTree([]).positions, []);
});

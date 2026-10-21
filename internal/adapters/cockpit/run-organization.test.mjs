import test from "node:test";
import assert from "node:assert/strict";
import { runNodes, managerUsage, exchangeFrames, exchangeRoute, pointAlong, activity } from "./web/js/run-organization-facts.mjs";

function fixture() {
  return { run: { id: "synthetic", binding: { managers: [{ manager: ".", execute: { profile: "writer" }, verify: { profile: "reviewer" }, maxHelpers: 2, helpers: [{ role: "worker", profile: "helper" }] }], checks: [{ name: "unit" }] } },
    current: "p2", passes: [{ id: "p2", purpose: "verify" }], ownerChannel: false,
    managers: [{ path: ".", label: "Root", depth: 0, state: "waiting", helpers: [], wait: { on: ["stock"], text: "result of stock" } }, { path: "stock", parent: ".", depth: 1, state: "working", turn: { role: "reviewer", short: "r1", readOnly: true }, helpers: [{ role: "worker", id: "h1" }] }],
    turns: [{ manager: ".", role: "manager", usage: { model: "alpha", costUSD: .1, providerTurns: 3 } }, { manager: ".", role: "reviewer", usage: { model: "beta", costUSD: .2, providerTurns: 1 } }, { manager: ".", role: "reviewer" }, { manager: "stock", role: "reviewer" }],
    findings: [{ owner: ".", kind: "suspected", closed: false }, { owner: ".", kind: "suggestion", closed: false }, { owner: ".", kind: "deviation", closed: true }],
    probes: [{ manager: ".", adverse: true }, { manager: ".", adverse: true, disposition: { as: "disproved" } }],
  };
}

test("recorded binding and current work stay separate from reported usage", () => {
  const view = fixture(), before = structuredClone(view), nodes = runNodes(view), root = nodes.find((n) => n.id === "."), stock = nodes.find((n) => n.id === "stock");
  assert.equal(root.state, "waiting"); assert.ok(root.badges.includes("Waits for stock"));
  assert.ok(root.badges.includes("1 suspicions")); assert.ok(root.badges.includes("1 suggestions"));
  assert.ok(root.badges.includes("1 adverse probes without accepted disposition"));
  assert.match(root.facts[1], /3 turns.*0.30 USD reported.*partial/);
  assert.ok(root.badges.includes("Models reported: alpha, beta"));
  assert.equal(root.reviewer.text, "reviewer · no active reviewer recorded");
  assert.equal(stock.reviewer.state, "working"); assert.match(stock.facts[0], /reviewer turn r1/);
  assert.ok(stock.badges.includes("Active: worker h1"));
  assert.deepEqual(managerUsage(view.turns, "."), { turns: 3, reported: 2, providerTurns: 4, costUSD: .30000000000000004, models: ["alpha", "beta"] });
  assert.deepEqual(view, before);
});

test("historical records never borrow today's profiles or invent helper limits", () => {
  const view = fixture(); delete view.run.binding;
  const root = runNodes(view).find((n) => n.id === ".");
  assert.match(root.reviewer.text, /Profile not recorded/);
  assert.ok(!root.badges.some((b) => b.startsWith("Helper limit:")));
  assert.equal(activity(view.managers[1]), "reviewer turn r1, read-only");
});

test("a clipped prefix has no later exchange or closed-Run owner channel", () => {
  const entries = [
    { seq: 1, link: { from: "stock", to: ".", channel: "submission" } },
    { seq: 2, link: { from: "@orchestrator", to: "stock", channel: "orchestrator" } },
    { seq: 3, link: { from: ".", to: "@user", channel: "owner" } },
  ];
  assert.deepEqual(exchangeFrames(entries, 1, 1, [0, 1, 2]).map((f) => f.entry.seq), [1]);
  assert.deepEqual(exchangeFrames(entries, 3, 2, [0, 1, 2]).map((f) => f.entry.seq), [1, 2]);
  assert.deepEqual(exchangeFrames(entries, 3, 2, [0, 1, 2], { ownerChannel: true }).map((f) => f.entry.seq), [1, 2, 3]);
});

test("the same paused clock gives identical progress; still replay is complete", () => {
  const entries = [{ seq: 1, link: { from: "stock", to: ".", channel: "submission" } }];
  const a = exchangeFrames(entries, 1, .3, [0], { animate: true }), b = exchangeFrames(entries, 1, .3, [0], { animate: true });
  assert.deepEqual(a, b); assert.ok(a[0].progress > 0 && a[0].progress < 1);
  assert.equal(exchangeFrames(entries, 1, .3, [0])[0].progress, 1);
});

test("submission goes upward, acceptance downward and control directly to its target", () => {
  const parent = { x: 100, y: 0, w: 244, h: 164 }, child = { x: 400, y: 222, w: 244, h: 164 };
  const up = exchangeRoute(child, parent, "submission"), down = exchangeRoute(parent, child, "agent"), control = exchangeRoute(parent, child, "orchestrator");
  assert.ok(up[0].y > up.at(-1).y); assert.ok(down[0].y < down.at(-1).y);
  assert.equal(control.length, 3); assert.deepEqual(pointAlong(control, 1), { ...control.at(-1), angle: 90 });
  for (const points of [up, down, control]) for (let i = 1; i < points.length; i++) assert.ok(points[i].x === points[i-1].x || points[i].y === points[i-1].y);
});

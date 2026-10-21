import test from "node:test";
import assert from "node:assert/strict";
import { observedNodes, profileText, browseHref, profilesHref } from "./web/js/organization-facts.js";
import { layoutTree, initialCollapse } from "./web/js/org-layout.js";

function manager(path, parent, depth) {
  return { path, parent, depth, label: path === "." ? "Root" : path.split("/").at(-1), definitions: [], areas: [], execute: { name: "author" }, verify: { name: "reviewer" } };
}

test("a one-Package observation has Owner, operations, Root and one department", () => {
  const observation = { mode: "declared", strategy: "direct", checks: [], managers: [manager(".", "", 0), { ...manager("library", ".", 1), definitions: [{ package: "library", id: "loans" }], areas: ["src/"] }] };
  const before = structuredClone(observation), nodes = observedNodes(observation);
  assert.deepEqual(nodes.map((n) => n.id), ["@user", "@orchestrator", ".", "library"]);
  assert.equal(nodes.find((n) => n.id === "library").parent, ".");
  assert.equal(nodes.find((n) => n.id === ".").parent, "@orchestrator");
  assert.equal(nodes.find((n) => n.id === "library").reviewer.text, "Verify: reviewer");
  assert.deepEqual(observation, before);
  assert.equal(layoutTree(nodes).positions.length, 4);
});

test("source Manager parents are kept in a large synthetic observation", () => {
  const observation = { mode: "resolved", strategy: "staged", checks: [{ name: "build" }], managers: [manager(".", "", 0), manager("product", ".", 1), ...Array.from({ length: 23 }, (_, i) => manager(`product/p${i}`, "product", 2))] };
  observation.managers[1].execute.model = "provider-model";
  observation.managers[1].team = { maxHelpers: 2, helpers: [] };
  const nodes = observedNodes(observation), product = nodes.find((n) => n.id === "product");
  assert.ok(product.facts[0].includes("provider-model"));
  assert.deepEqual(product.badges, ["Resolved", "2 active helpers at most"]);
  assert.equal(layoutTree(nodes, initialCollapse(nodes)).positions.length, 4);
});

test("absent declaration remains absent and navigation preserves exact names", () => {
  assert.equal(profileText({}), "Not declared");
  assert.equal(profileText({ name: "p" }), "p");
  assert.equal(profileText({ name: "p", model: "m" }), "p Â· m");
  assert.equal(profilesHref("."), "#/settings?section=profiles");
  const pkg = "a/b + Ã¼";
  assert.equal(new URLSearchParams(profilesHref(pkg).split("?")[1]).get("pkg"), pkg);
  const params = new URLSearchParams(browseHref(pkg, "rule + 1").split("?")[1]);
  assert.equal(params.get("pkg"), pkg); assert.equal(params.get("def"), "rule + 1");
});

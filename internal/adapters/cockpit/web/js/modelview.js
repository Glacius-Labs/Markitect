// The model browser: the checked canonical model as the project observation
// prepared it. Packages on the left, their Definitions in the middle, the
// chosen Definition on the right with its body (links the model check located
// open their target), its fields, the files the registry says realize it and
// the Definitions that link to it. It reads; changes go through a model
// session or by hand.

import { api } from "./api.js";
import { h, clear, when } from "./dom.js";
import { store } from "./store.js";

let cache = null; // { observedAt, data }

export function renderModelView(content, route) {
  const tree = h("div", { class: "pane tree-pane" });
  const defs = h("div", { class: "pane defs" });
  const detail = h("div", { class: "pane detail" });
  const view = h("div", { class: `modelview${route.id ? "" : " showdefs"}` }, tree, defs, detail);
  content.append(view);
  clear(detail).append(h("p", { class: "void", role: "status" }, "Reading the model…"));
  const show = (data) => {
    const m = index(data);
    drawLists(tree, defs, m, route);
    drawDetail(view, detail, m, route);
  };
  const load = () => api.model().then((data) => {
    cache = { observedAt: data.observedAt, data };
    if (view.isConnected) show(data);
  }, (err) => {
    if (!view.isConnected) return;
    const waiting = /being observed/.test(err.message);
    clear(detail).append(h("p", { class: "void", role: waiting ? "status" : "alert" }, h("strong", {}, waiting ? "Observing the repository" : "The model is not readable"), waiting ? "The structural preparation of the working tree runs; the model appears when it ends." : err.message));
    if (waiting) setTimeout(() => view.isConnected && load(), 2500);
  });
  // A newer observation (after Observe again or an accepted proposal) is read
  // again; otherwise the last one serves.
  const observed = store.project?.observedAt;
  if (cache && (!observed || cache.observedAt === observed)) show(cache.data);
  else load();
  const unsubscribe = store.subscribe((s) => {
    if (!view.isConnected) return unsubscribe();
    if (cache && s.project?.observedAt && s.project.observedAt !== cache.observedAt) load();
  });
}

// index prepares the model for lookups: every Definition with its scope and
// by key.
function index(data) {
  const scopes = data.model.models;
  const all = scopes.flatMap((s) => s.definitions.map((d) => ({ ...d, scope: s })));
  const byKey = new Map(all.map((d) => [`${d.package}/${d.id}`, d]));
  return { data, scopes, all, byKey };
}

function href(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const text = q.toString();
  return `#/model${text ? `?${text}` : ""}`;
}

// drawLists draws the Packages and the Definitions of the chosen Package or
// of the filter. The filter redraws only these two panes.
function drawLists(tree, defs, m, route) {
  const query = (route.q || "").trim().toLowerCase();
  const filter = h("input", { type: "search", placeholder: "Filter Definitions  /", value: route.q || "", "aria-label": "Filter Definitions" });
  filter.addEventListener("input", () => {
    route.q = filter.value;
    history.replaceState(null, "", href({ pkg: route.pkg, def: route.id, q: route.q }));
    drawLists(tree, defs, m, route);
    const f = tree.querySelector("input");
    f.focus();
    f.setSelectionRange(f.value.length, f.value.length);
  });
  const list = h("ul", { class: "tree" });
  list.append(h("li", {}, treeLink(href({ q: route.q }), !route.pkg, "All Definitions", m.all.filter((d) => matches(d, query)).length)));
  for (const scope of m.scopes) {
    list.append(h("li", { class: "group" }, scope.local ? `This project: ${scope.id}` : `Dependency: ${scope.id}`));
    const pkgs = [...new Set([...scope.packages, ...scope.definitions.map((d) => d.package)])].sort();
    for (const pkg of pkgs) {
      const n = scope.definitions.filter((d) => d.package === pkg && matches(d, query)).length;
      if (query && !n) continue;
      list.append(h("li", {}, treeLink(href({ pkg, q: route.q }), route.pkg === pkg, pkg, n, depth(pkg))));
    }
  }
  clear(tree).append(h("div", { class: "filter" }, filter), list);

  const shown = m.all.filter((d) => (!route.pkg || d.package === route.pkg) && matches(d, query));
  clear(defs).append(h("div", { class: "panehead" }, h("h2", {}, route.pkg || (query ? "Matching Definitions" : "All Definitions")), h("span", { class: "coord" }, String(shown.length))));
  if (!shown.length) {
    defs.append(h("p", { class: "void" }, query ? `No Definition matches “${route.q}”.` : "This Package holds no Definition."));
    return;
  }
  defs.append(h("ul", { class: "rows" }, shown.map((d) => h("li", {},
    h("a", { class: "row", href: href({ pkg: d.package, def: d.id, q: route.q }), "aria-current": String(route.pkg === d.package && route.id === d.id) },
      h("span", { class: "kind" }, kindName(d.kind)),
      h("span", { class: "title" }, d.title || d.id),
      h("span", { class: "meta" }, route.pkg ? "" : d.package),
      h("span", { class: "sub" }, d.title ? d.id : firstSentence(d)))))));
}

function drawDetail(view, detail, m, route) {
  const d = route.id ? m.byKey.get(`${route.pkg}/${route.id}`) : null;
  view.classList.toggle("showdefs", !d);
  if (!d) {
    clear(detail).append(overviewPanel(m, route.id ? `${route.pkg}/${route.id}` : null));
    return;
  }
  const key = `${d.package}/${d.id}`;
  const registryRead = m.data.model.registryStatus === "passed";
  const realized = m.data.model.registry.filter((r) => r.realizes.includes(key));
  const linkedFrom = m.all.filter((x) => x.links.some((l) => `${l.package}/${l.id}` === key));
  const kindDef = m.byKey.get(d.kind);
  clear(detail).append(
    h("div", { class: "defhead" },
      h("div", { class: "path coord" }, d.scope.local ? d.package : `${d.package} in dependency ${d.scope.id}`),
      h("h2", {}, d.title || d.id),
      h("div", { class: "line" },
        kindDef ? h("a", { class: "kind", href: href({ pkg: kindDef.package, def: kindDef.id }) }, kindName(d.kind)) : h("span", { class: "kind" }, kindName(d.kind)),
        d.title ? h("span", { class: "digest" }, d.id) : null,
        h("span", { class: "digest faint", title: "Source file" }, d.source)),
      d.scope.status !== "passed" ? h("p", { class: "refusal", style: { marginTop: "14px" } }, `This model's check did not pass (${d.scope.status}); the Definition is shown as read.`) : null),
    h("div", { class: "defbody" },
      h("section", {}, h("h3", {}, "Body"), h("p", { class: "text" }, bodyView(d, m.byKey))),
      d.spec.length ? h("section", {}, h("h3", {}, "Fields"), h("table", { class: "table" }, h("tbody", {}, d.spec.map((f) => h("tr", {}, h("th", { scope: "row", style: { width: "1%", whiteSpace: "nowrap" } }, f.name), h("td", { class: "digest" }, f.value)))))) : null,
      h("section", {}, h("h3", {}, `Realized by ${realized.length} registered file${realized.length === 1 ? "" : "s"}`),
        registryRead ? null : h("p", { class: "refusal", style: { marginBottom: "10px" } }, `The registry was not read completely (${m.data.model.registryStatus}); files may be missing from this list.`),
        realized.length ? h("table", { class: "table" }, h("thead", {}, h("tr", {}, h("th", {}, "Path"), h("th", {}, "Form"), h("th", {}, "Why"))),
          h("tbody", {}, realized.map((r) => h("tr", {}, h("td", { class: "digest path" }, r.path), h("td", {}, r.form), h("td", {}, r.why, h("div", { class: "sub digest" }, r.shard))))))
          : h("p", { class: "note", style: { margin: 0 } }, registryRead ? "No registered file names this Definition." : "No file read from the registry names this Definition.")),
      h("section", {}, h("h3", {}, `Linked from ${linkedFrom.length} Definition${linkedFrom.length === 1 ? "" : "s"}`),
        linkedFrom.length ? h("ul", { class: "rows", style: { border: "1px solid var(--rule-2)" } }, linkedFrom.map((x) => h("li", {},
          h("a", { class: "row", href: href({ pkg: x.package, def: x.id }) }, h("span", { class: "kind" }, kindName(x.kind)), h("span", { class: "title" }, x.title || x.id), h("span", { class: "meta" }, x.package)))))
          : h("p", { class: "note", style: { margin: 0 } }, "No Definition links to this one."))));
  detail.scrollTop = 0;
}

function treeLink(to, current, text, n, level = 0) {
  return h("a", { href: to, "aria-current": String(current), title: text, style: level ? { paddingLeft: `${16 + level * 14}px` } : null }, h("span", {}, level ? text.split("/").pop() : text), h("span", { class: "count" }, String(n)));
}

function depth(pkg) {
  return pkg.split("/").length - 1;
}

function plainText(d) {
  return (d.segments || [{ text: d.body }]).map((s) => s.text).join("");
}

function matches(d, query) {
  if (!query) return true;
  return d.id.toLowerCase().includes(query) || (d.title || "").toLowerCase().includes(query) || plainText(d).toLowerCase().includes(query) || d.package.toLowerCase().includes(query);
}

function kindName(kind) {
  return kind.split("/").pop();
}

function firstSentence(d) {
  const text = plainText(d).replace(/\s+/g, " ").trim();
  const end = text.search(/[.!?](\s|$)/);
  const sentence = end > 0 ? text.slice(0, end + 1) : text;
  return sentence.length > 140 ? `${sentence.slice(0, 139)}…` : sentence;
}

// bodyView renders the body from the segments the cockpit served: text, and
// each link the model check located with its target. Nothing in the body is
// parsed here, so no text is shown as a link the model does not hold.
function bodyView(d, byKey) {
  return (d.segments || [{ text: d.body }]).map((seg) => {
    if (!seg.link) return seg.text;
    const target = `${seg.link.package}/${seg.link.id}`;
    const t = byKey.get(target);
    return t
      ? h("a", { class: "deflink", href: href({ pkg: t.package, def: t.id }), title: target }, seg.text)
      : h("span", { class: "deflink external", title: `${target} is not in the browsed models` }, seg.text);
  });
}

function overviewPanel(m, missing) {
  const local = m.scopes.find((s) => s.local);
  const total = m.scopes.reduce((n, s) => n + s.definitions.length, 0);
  const kinds = {};
  for (const s of m.scopes) for (const d of s.definitions) kinds[kindName(d.kind)] = (kinds[kindName(d.kind)] || 0) + 1;
  const failed = m.scopes.filter((s) => s.status !== "passed");
  return h("div", { class: "panebody blueprint", style: { minHeight: "100%" } },
    missing ? h("p", { class: "refusal", role: "alert" }, `The model holds no Definition ${missing}.`) : null,
    h("div", { class: "pagehead" }, h("div", {}, h("span", { class: "label" }, "Canonical model"), h("h2", {}, local ? local.id : "Model"),
      h("p", { class: "lead" }, `Observed ${when(m.data.observedAt)} from the working tree at `, h("span", { class: "digest" }, (m.data.head || "").slice(0, 12)), ". Choose a Package or a Definition, or filter with /."))),
    failed.length ? h("p", { class: "refusal" }, `The check of ${failed.map((s) => s.id).join(", ")} did not pass; what the observation read is shown.`) : null,
    m.data.model.registryStatus !== "passed" ? h("p", { class: "refusal" }, `The registry was not read completely (${m.data.model.registryStatus}).`) : null,
    h("div", { class: "stat", style: { margin: "8px 0 28px" } },
      h("div", {}, h("b", {}, String(total)), h("span", {}, "Definitions")),
      h("div", {}, h("b", {}, String(m.scopes.reduce((n, s) => n + s.packages.length, 0))), h("span", {}, "Packages")),
      h("div", {}, h("b", {}, String(m.data.model.registry.length)), h("span", {}, "Registered files")),
      h("div", {}, h("b", {}, String(m.scopes.length)), h("span", {}, m.scopes.length === 1 ? "Model" : "Models"))),
    h("h3", { class: "label", style: { margin: "0 0 10px" } }, "Kinds"),
    h("table", { class: "table", style: { maxWidth: "520px", border: "1px solid var(--rule-2)" } }, h("tbody", {},
      Object.entries(kinds).sort((a, b) => b[1] - a[1]).map(([k, n]) => h("tr", {}, h("td", {}, k), h("td", { class: "digest", style: { textAlign: "right" } }, String(n)))))),
    local?.digest ? h("p", { class: "note", style: { marginTop: "22px" } }, h("span", { class: "label" }, "Digest "), h("span", { class: "digest" }, local.digest)) : null);
}

// "/" focuses the filter while the model browser is open.
document.addEventListener("keydown", (e) => {
  if (e.key !== "/" || e.target.closest?.("input, textarea, select")) return;
  const f = document.querySelector(".modelview .filter input");
  if (f) {
    e.preventDefault();
    f.focus();
  }
});

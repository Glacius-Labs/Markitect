// The model browser: the checked canonical model as the project observation
// prepared it. Packages on the left, their Definitions in the middle, the
// chosen Definition on the right with its body (links open their target), its
// fields, the files the registry says realize it and the Definitions that link
// to it. It reads; changes go through a model session or by hand.

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
  clear(detail).append(h("p", { class: "empty", role: "status" }, "Reading the model…"));
  const load = () => api.model().then((data) => {
    cache = { observedAt: data.observedAt, data };
    if (view.isConnected) draw(view, tree, defs, detail, data, route);
  }, (err) => {
    if (!view.isConnected) return;
    const waiting = /being observed/.test(err.message);
    clear(detail).append(h("p", { class: "empty", role: waiting ? "status" : "alert" }, h("strong", {}, waiting ? "Observing the repository" : "The model is not readable"), waiting ? "The structural preparation of the working tree runs; the model appears when it ends." : err.message));
    if (waiting) setTimeout(() => view.isConnected && load(), 2500);
  });
  // A newer observation (after Observe again or an accepted proposal) is read
  // again; otherwise the last one serves.
  const observed = store.project?.observedAt;
  if (cache && (!observed || cache.observedAt === observed)) draw(view, tree, defs, detail, cache.data, route);
  else load();
  const unsubscribe = store.subscribe((s) => {
    if (!view.isConnected) return unsubscribe();
    if (cache && s.project?.observedAt && s.project.observedAt !== cache.observedAt) load();
  });
}

function href(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const text = q.toString();
  return `#/model${text ? `?${text}` : ""}`;
}

function draw(view, tree, defs, detail, data, route) {
  const scopes = data.model.models;
  const all = scopes.flatMap((s) => s.definitions.map((d) => ({ ...d, scope: s })));
  const byKey = new Map(all.map((d) => [`${d.package}/${d.id}`, d]));
  const query = (route.q || "").trim().toLowerCase();

  // ---- packages ----
  const filter = h("input", { type: "search", placeholder: "Filter Definitions  (/)", value: route.q || "", "aria-label": "Filter Definitions" });
  filter.addEventListener("input", () => {
    const next = href({ pkg: route.pkg, def: route.id, q: filter.value });
    history.replaceState(null, "", next);
    route.q = filter.value;
    draw(view, tree, defs, detail, data, route);
    const f = tree.querySelector("input");
    f.focus();
    f.setSelectionRange(f.value.length, f.value.length);
  });
  const list = h("ul", { class: "tree" });
  const count = (pkg, scope) => scope.definitions.filter((d) => d.package === pkg && matches(d, query)).length;
  list.append(h("li", {}, treeLink(href({ q: route.q }), !route.pkg, "All Definitions", all.filter((d) => matches(d, query)).length)));
  for (const scope of scopes) {
    list.append(h("li", { class: "group" }, scope.local ? `This project: ${scope.id}` : `Dependency: ${scope.id}`));
    const pkgs = [...new Set([...scope.packages, ...scope.definitions.map((d) => d.package)])].sort();
    for (const pkg of pkgs) {
      const n = count(pkg, scope);
      if (query && !n) continue;
      list.append(h("li", {}, treeLink(href({ pkg, q: route.q }), route.pkg === pkg, pkg, n, depth(pkg))));
    }
  }
  clear(tree).append(h("div", { class: "filter" }, filter), list);

  // ---- Definitions of the chosen Package or all matching ----
  const shown = all.filter((d) => (!route.pkg || d.package === route.pkg) && matches(d, query));
  clear(defs).append(h("div", { class: "panehead" }, h("h2", {}, route.pkg || (query ? "Matching Definitions" : "All Definitions")), h("span", { class: "note" }, String(shown.length))));
  if (!shown.length) {
    defs.append(h("p", { class: "empty" }, query ? `No Definition matches “${route.q}”.` : "This Package holds no Definition."));
  } else {
    defs.append(h("ul", { class: "rows" }, shown.map((d) => h("li", {},
      h("a", { class: "row", href: href({ pkg: d.package, def: d.id, q: route.q }), "aria-current": String(route.pkg === d.package && route.id === d.id) },
        h("span", { class: "kind" }, kindName(d.kind)),
        h("span", { class: "title" }, d.title || d.id),
        h("span", { class: "meta" }, route.pkg ? "" : d.package),
        h("span", { class: "sub" }, d.title ? d.id : firstSentence(d.body)))))));
  }

  // ---- the chosen Definition ----
  const d = route.id ? byKey.get(`${route.pkg}/${route.id}`) : null;
  view.classList.toggle("showdefs", !d);
  if (!d) {
    clear(detail).append(overviewPanel(data, scopes, route.id ? `${route.pkg}/${route.id}` : null));
    return;
  }
  const key = `${d.package}/${d.id}`;
  const realized = data.model.registry.filter((r) => r.realizes.includes(key));
  const linkedFrom = all.filter((x) => x.links.some((l) => `${l.package}/${l.id}` === key));
  const kindDef = byKey.get(d.kind);
  clear(detail).append(
    h("div", { class: "defhead" },
      h("div", { class: "path" }, `Package ${d.package} in ${d.scope.local ? "this project's model" : `the dependency ${d.scope.id}`}`),
      h("h2", {}, d.title || d.id),
      h("div", { class: "line" },
        kindDef ? h("a", { class: "kind", href: href({ pkg: kindDef.package, def: kindDef.id }) }, kindName(d.kind)) : h("span", { class: "kind" }, kindName(d.kind)),
        h("span", { class: "digest" }, d.id),
        h("span", { class: "digest faint", title: "Source file" }, d.source))),
    h("div", { class: "defbody" },
      h("section", {}, h("h3", {}, "Body"), h("p", { class: "text" }, bodyWithLinks(d, byKey))),
      d.spec.length ? h("section", {}, h("h3", {}, "Fields"), h("table", { class: "table" }, h("tbody", {}, d.spec.map((f) => h("tr", {}, h("th", { scope: "row", style: { width: "1%", whiteSpace: "nowrap" } }, f.name), h("td", { class: "digest" }, f.value)))))) : null,
      h("section", {}, h("h3", {}, `Realized by ${realized.length} registered file${realized.length === 1 ? "" : "s"}`),
        realized.length ? h("table", { class: "table" }, h("thead", {}, h("tr", {}, h("th", {}, "Path"), h("th", {}, "Form"), h("th", {}, "Why"))),
          h("tbody", {}, realized.map((r) => h("tr", {}, h("td", { class: "digest path" }, r.path), h("td", {}, r.form), h("td", {}, r.why, h("div", { class: "sub digest" }, r.shard))))))
          : h("p", { class: "note", style: { margin: 0 } }, "No registered file names this Definition.")),
      h("section", {}, h("h3", {}, `Linked from ${linkedFrom.length} Definition${linkedFrom.length === 1 ? "" : "s"}`),
        linkedFrom.length ? h("ul", { class: "rows", style: { border: "1px solid var(--rule)", borderRadius: "6px", background: "var(--plate)" } }, linkedFrom.map((x) => h("li", {},
          h("a", { class: "row", href: href({ pkg: x.package, def: x.id }) }, h("span", { class: "kind" }, kindName(x.kind)), h("span", { class: "title" }, x.title || x.id), h("span", { class: "meta" }, x.package)))))
          : h("p", { class: "note", style: { margin: 0 } }, "No Definition links to this one."))));
  detail.scrollTop = 0;
}

function treeLink(to, current, text, n, level = 0) {
  return h("a", { href: to, "aria-current": String(current), title: text, style: level ? { paddingLeft: `${14 + level * 12}px` } : null }, h("span", {}, level ? text.split("/").pop() : text), h("span", { class: "count" }, String(n)));
}

function depth(pkg) {
  return pkg.split("/").length - 1;
}

function matches(d, query) {
  if (!query) return true;
  return d.id.toLowerCase().includes(query) || (d.title || "").toLowerCase().includes(query) || d.body.toLowerCase().includes(query) || d.package.toLowerCase().includes(query);
}

function kindName(kind) {
  return kind.split("/").pop();
}

function firstSentence(body) {
  const text = body.replace(/\[\[([^\]|]+)\|([^\]]+)\]\]/g, "$2").replace(/\[\[([^\]]+)\]\]/g, (_, t) => t.split(".").pop()).replace(/\s+/g, " ").trim();
  const end = text.search(/[.!?](\s|$)/);
  const sentence = end > 0 ? text.slice(0, end + 1) : text;
  return sentence.length > 140 ? `${sentence.slice(0, 139)}…` : sentence;
}

// bodyWithLinks renders the body text with each [[link]] as a button to its
// target. The n-th link of the text names the n-th recorded target.
function bodyWithLinks(d, byKey) {
  const out = [];
  const re = /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g;
  let last = 0;
  let i = 0;
  for (let m; (m = re.exec(d.body));) {
    out.push(d.body.slice(last, m.index));
    const target = d.links[i++];
    const shown = m[2] || m[1].split(".").pop();
    if (target) {
      const t = byKey.get(`${target.package}/${target.id}`);
      out.push(t
        ? h("a", { class: "deflink", href: href({ pkg: t.package, def: t.id }), title: `${target.package}/${target.id}` }, shown)
        : h("span", { class: "deflink external", title: `${target.package}/${target.id} is not in the browsed models` }, shown));
    } else {
      out.push(m[0]);
    }
    last = m.index + m[0].length;
  }
  out.push(d.body.slice(last));
  return out;
}

function overviewPanel(data, scopes, missing) {
  const local = scopes.find((s) => s.local);
  const total = scopes.reduce((n, s) => n + s.definitions.length, 0);
  const kinds = {};
  for (const s of scopes) for (const d of s.definitions) kinds[kindName(d.kind)] = (kinds[kindName(d.kind)] || 0) + 1;
  return h("div", { class: "panebody" },
    missing ? h("p", { class: "refusal", role: "alert" }, `The model holds no Definition ${missing}.`) : null,
    h("div", { class: "pagehead" }, h("div", {}, h("h2", {}, local ? `Model ${local.id}` : "Model"),
      h("p", { class: "lead" }, `Observed ${when(data.observedAt)} from the working tree at `, h("span", { class: "digest" }, (data.head || "").slice(0, 12)), ". Choose a Package or a Definition, or filter with /."))),
    h("div", { class: "stat" },
      h("div", {}, h("b", {}, String(total)), h("span", {}, "Definitions")),
      h("div", {}, h("b", {}, String(scopes.reduce((n, s) => n + s.packages.length, 0))), h("span", {}, "Packages")),
      h("div", {}, h("b", {}, String(data.model.registry.length)), h("span", {}, "Registered files")),
      h("div", {}, h("b", {}, String(scopes.length)), h("span", {}, scopes.length === 1 ? "Model" : "Models"))),
    h("h3", { style: { fontSize: "var(--s0)", color: "var(--ink-2)", margin: "22px 0 8px" } }, "Kinds"),
    h("table", { class: "table", style: { maxWidth: "520px", background: "var(--plate)", border: "1px solid var(--rule)" } }, h("tbody", {},
      Object.entries(kinds).sort((a, b) => b[1] - a[1]).map(([k, n]) => h("tr", {}, h("td", {}, k), h("td", { style: { textAlign: "right" } }, String(n)))))),
    local?.digest ? h("p", { class: "note" }, "Canonical model digest ", h("span", { class: "digest" }, local.digest)) : null);
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

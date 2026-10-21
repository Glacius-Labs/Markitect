import { api } from "./api.js";
import { h, append, clear, pagehead, about, empty, shortDigest } from "./dom.js";
import { store } from "./store.js";
import { OrgChart } from "./orgchart.js";
import { observedNodes, profileText, browseHref, profilesHref } from "./organization-facts.js";

// Declared data is immediately available from the project cache; Runtime
// capture happens only after the owner requests it. The page never edits.
export function renderOrganization(content) {
  let closed = false, reading = false, refreshPending = false, timer, data, selected = ".", wantResolved = false, drawn = "";
  let observedAt = store.project?.observedAt;
  const resolve = h("button", { class: "btn", type: "button", disabled: true, onclick: () => { wantResolved = true; load(true); } }, "Resolve profiles");
  const declared = h("button", { class: "btn quiet", type: "button", hidden: true, onclick: () => { wantResolved = false; draw(); } }, "Show declarations");
  const state = h("div", { class: "organization-state", role: "status" });
  const problem = h("div", { class: "refusal", role: "alert" });
  const stage = h("section", { class: "organization-stage", "aria-label": "Organization chart" });
  const detail = h("aside", { class: "organization-detail plate", "aria-label": "Selected responsibility" });
  const page = h("div", { class: "page organization-page" },
    pagehead("Project organization", "Who leads, what each department owns, and who reviews independently.", {
      actions: [declared, resolve], about: about("How this works", h("p", {}, "Declarations come from the working-tree observation. Resolve profiles applies the product defaults and binds the Runtime profiles from a clean committed source; it can take up to two minutes. No Run, Agent or project check starts."), h("p", {}, "Change profiles and the project-wide team in Settings through the existing proposal flow. The chart itself edits nothing.")) }),
    state, problem, h("div", { class: "organization-split" }, stage, detail));
  content.append(page);
  const chart = new OrgChart(stage, { onSelect: (id) => { selected = id; draw(); } });

  function draw() {
    if (closed || !data) return;
    const v = wantResolved && data.resolved ? data.resolved : data.declared;
    resolve.disabled = !data.resolvable || data.resolving || reading;
    resolve.textContent = data.resolving ? "Resolving profilesâ€¦" : "Resolve profiles";
    resolve.title = data.resolvable ? "Read the committed settings with the configured Runtime profile" : "A Runtime profile and run configuration are needed to resolve profiles";
    declared.hidden = !wantResolved;
    const status = [data.resolving ? "Resolving the committed source; declarations stay visible until it is ready." : "", v ? `${v.mode === "resolved" ? "Resolved from committed source" : "Declared in the working-tree observation"} Â· ${v.head ? v.head.slice(0, 12) : "head not observed"} Â· model ${shortDigest(v.modelDigest) || "not checked"}` : data.observing ? "Observing the repositoryâ€¦" : "No checked organization is available.", !data.resolvable ? "Runtime resolution is not configured." : ""].filter(Boolean).join(" ");
    if (state.textContent !== status) state.textContent = status;
    problem.textContent = [data.error, v?.error].filter(Boolean).join(" ");
    if (!v?.managers.length) {
      chart.el.hidden = true;
      append(clear(detail), [empty("Organization unavailable", data.observing ? "The project observation is still running." : "Inspect the structural check and the model before using its organization.")]);
      return;
    }
    chart.el.hidden = false;
    const signature = JSON.stringify([v, selected]);
    if (signature === drawn) return;
    drawn = signature;
    if (!v.managers.some((m) => m.path === selected) && !selected.startsWith("@")) selected = ".";
    chart.update(observedNodes(v), { key: v.modelID, selection: selected });
    drawDetail(detail, v, selected);
  }

  async function load(requestResolution = false) {
    if (closed) return;
    if (reading) { refreshPending = true; return; }
    reading = true; draw(); clearTimeout(timer);
    try {
      const answer = await api.organization(requestResolution);
      if (closed) return;
      data = answer;
      reading = false; draw();
      if (refreshPending) { refreshPending = false; load(); }
      else if (data.resolving || data.observing) timer = setTimeout(() => load(), 1500);
    } catch (err) {
      if (!closed) {
        reading = false; resolve.disabled = !data?.resolvable; problem.textContent = err.message;
        if (refreshPending) { refreshPending = false; load(); }
      }
    }
  }
  const unsubscribe = store.subscribe((s) => {
    if (closed) return;
    if (s.project?.observedAt && s.project.observedAt !== observedAt) { observedAt = s.project.observedAt; load(); }
  });
  load();
  return () => { closed = true; clearTimeout(timer); unsubscribe(); chart.destroy(); };
}

function drawDetail(el, v, selected) {
  clear(el);
  if (selected === "@user") {
    append(el, [h("h3", {}, "Owner"), h("p", {}, "The project's principal maintains intent in the model, starts Runs and decides proposals. This card is context, not an external message channel."), h("a", { href: "#/decisions" }, "Open decisions")]);
    return;
  }
  if (selected === "@orchestrator") {
    append(el, [h("h3", {}, "Orchestrator"), h("p", {}, "The deterministic controller runs checks, routes and accepts submissions, and keeps the budget. It can address every Manager directly."), h("p", {}, `Run-wide strategy: ${v.strategy || "not declared"}`), h("h4", {}, "Declared checks"),
      v.checks.length ? h("ul", {}, v.checks.map((c) => h("li", {}, h("strong", {}, c.name), h("p", {}, `${c.profile}: ${c.args.join(" ")} in ${c.directory}, ${c.timeout}`), c.requiredBy ? definitionLink(c.requiredBy) : null))) : h("p", { class: "note" }, "No checks declared.")]);
    return;
  }
  const m = v.managers.find((m) => m.path === selected);
  if (!m) return;
  append(el, [h("h3", {}, m.path === "." ? "Root" : m.path), h("p", { class: "note" }, `${v.mode === "resolved" ? "Resolved" : "Declared"} profiles and responsibilities`),
    h("h4", {}, "Profiles"), profile("Execute", m.execute), profile("Independent Verify", m.verify),
    h("a", { class: "btn small", href: profilesHref(m.path) }, "Change profiles"),
    h("h4", {}, "Own Definitions"), m.definitions.length ? h("ul", {}, m.definitions.map((d) => h("li", {}, h("a", { href: browseHref(d.package, d.id) }, d.id)))) : h("p", { class: "note" }, m.path === "." ? "Root has no local Definitions; it coordinates the complete project." : "This Package has no own Definitions."),
    h("h4", {}, "Owned areas"), v.error && !v.ownership ? h("p", { class: "note" }, "Validated areas are unavailable; inspect the preparation error above.") : !v.ownership ? h("p", { class: "note" }, "No ownership layout is declared. Registry assignments are separate.") : m.areas.length ? h("ul", {}, m.areas.map((area) => h("li", {}, area))) : h("p", { class: "note" }, "No artifact area assigned to this Manager."),
    h("h4", {}, "Project-wide team"), h("p", { class: "note" }, "This team applies to every Manager. Per-Package overrides change only Execute and Verify profiles."),
    m.team ? h("div", {}, h("p", {}, `At most ${m.team.maxHelpers} helpers active per Manager.`), h("ul", {}, m.team.helpers.map((helper) => h("li", {}, h("strong", {}, helper.role), ` Â· ${helper.writes ? "may write delegated files" : "read-only"}`, h("p", {}, profileText(helper.profile, "Manager's Execute profile")))))) : h("p", { class: "note" }, "No team declaration. Resolve profiles to inspect the product default."),
    h("a", { class: "btn small", href: "#/settings?section=team" }, "Change the project-wide team")]);
}

function profile(role, p) {
  return h("div", { class: "organization-profile" }, h("strong", {}, role), h("p", {}, profileText(p)), p?.identity ? h("p", { class: "digest", title: p.identity }, p.identity) : null,
    p?.source ? about("Selection source", h("p", {}, p.source.file, p.source.field ? ` ${p.source.field}` : ""), h("p", { class: "digest" }, p.source.digest)) : null);
}

function definitionLink(identity) {
  const cut = identity.lastIndexOf("/");
  return cut > 0 ? h("a", { href: browseHref(identity.slice(0, cut), identity.slice(cut + 1)) }, identity) : h("span", {}, identity);
}

// Presentation of explicit organization observations; no Package/ownership rules.
export function profileText(profile, absent = "Not declared") {
  if (!profile?.name) return absent;
  return profile.name + (profile.model ? ` · ${profile.model}` : "");
}

export function observedNodes(organization) {
  const mode = organization.mode === "resolved" ? "Resolved" : "Declared";
  const nodes = [
    { id: "@user", parent: null, kind: "owner", title: "Project principal", name: "Owner", initial: "O", facts: ["Intent in the model", "Starts runs and decides proposals"] },
    { id: "@orchestrator", parent: "@user", kind: "orchestrator", title: "Deterministic operations", name: "Orchestrator", initial: "M", facts: [`${organization.checks?.length || 0} declared checks`, `Strategy: ${organization.strategy || "not declared"}`] },
  ];
  for (const m of organization.managers || []) {
    nodes.push({ id: m.path, parent: m.path === "." ? "@orchestrator" : m.parent,
      kind: "manager", managerDepth: m.depth, title: m.path === "." ? "Project lead" : "Package Manager", name: m.label || m.path,
      facts: [`Execute: ${profileText(m.execute)}`, `${m.definitions.length} Definitions · ${m.areas.length} owned areas`],
      badges: [mode, m.team ? `${m.team.maxHelpers} active helpers at most` : "Team not declared"],
      reviewer: { text: `Verify: ${profileText(m.verify)}` },
    });
  }
  return nodes;
}

export function browseHref(pkg, id) {
  const params = new URLSearchParams({ pkg });
  if (id) params.set("def", id);
  return `#/model?${params}`;
}

export function profilesHref(pkg) { return pkg === "." ? "#/settings?section=profiles" : `#/settings?${new URLSearchParams({ pkg })}`; }

import { passTitle } from "./runfacts.js";

const label = (id) => id === "." ? "Root" : id === "@orchestrator" ? "Orchestrator" : id === "@user" ? "Owner" : id;

export function managerUsage(turns, manager) {
  const own = turns.filter((t) => t.manager === manager), reported = own.filter((t) => t.usage);
  return { turns: own.length, reported: reported.length,
    providerTurns: reported.reduce((n, t) => n + (t.usage.providerTurns || 0), 0),
    costUSD: reported.reduce((n, t) => n + (t.usage.costUSD || 0), 0),
    models: [...new Set(reported.map((t) => t.usage.model).filter(Boolean))],
  };
}

export function runNodes(view) {
  const pass = view.passes.find((p) => p.id === view.current), binding = view.run.binding;
  const bound = new Map((binding?.managers || []).map((m) => [m.manager, m]));
  const nodes = [
    { id: "@user", parent: null, kind: "owner", title: "Project principal · static context", name: "Owner", initial: "O", facts: ["Model changes happen outside the Run", view.ownerChannel ? "Historical exchanges recorded" : "Closed Run · no external channel"] },
    { id: "@orchestrator", parent: "@user", kind: "orchestrator", title: "Deterministic operations", name: "Orchestrator", initial: "M", facts: [pass ? `${passTitle(pass.purpose)} ${pass.id}` : "No Pass recorded at this boundary", binding?.checks ? `${binding.checks.length} bound checks` : "Checks not recorded in a binding"] },
  ];
  for (const m of view.managers) {
    const b = bound.get(m.path), usage = managerUsage(view.turns, m.path);
    const observations = view.findings.filter((f) => f.owner === m.path && !f.closed);
    const badges = [];
    if (m.wait) badges.push(m.wait.outdated ? `Wait outdated${m.wait.on?.length ? `; needs ${m.wait.on.map(label).join(", ")}` : "; current reason not recorded"}` : `Waits for ${m.wait.on?.length ? m.wait.on.map(label).join(", ") : m.wait.text}`);
    for (const [kind, title] of [["deviation", "deviations"], ["suspected", "suspicions"], ["model", "model problems"], ["suggestion", "suggestions"]]) {
      const n = observations.filter((f) => f.kind === kind).length; if (n) badges.push(`${n} ${title}`);
    }
    for (const helper of m.helpers || []) badges.push(`Active: ${helper.role} ${helper.id}`);
    const adverse = view.probes.filter((p) => p.manager === m.path && p.adverse && (!p.disposition || p.disposition.pending)).length;
    if (adverse) badges.push(`${adverse} adverse probes without accepted disposition`);
    if (usage.models.length) badges.push(`Models reported: ${usage.models.join(", ")}`);
    if (b?.maxHelpers != null) badges.push(`Helper limit: ${b.maxHelpers}`);
    const current = m.turn ? `${m.turn.role} turn ${m.turn.short}` : m.state ? m.state : "Not started";
    nodes.push({ id: m.path, parent: m.path === "." ? "@orchestrator" : m.parent, kind: "manager", managerDepth: m.depth,
      title: m.path === "." ? "Root Manager" : "Package Manager", name: m.label || m.path, state: m.state,
      facts: [`${pass ? `${passTitle(pass.purpose)} ${pass.id} · ` : ""}${current}`, `${usage.turns} turns · ${usage.costUSD > 0 ? `${usage.costUSD.toFixed(2)} USD reported` : "cost not reported"}${usage.reported < usage.turns ? " (partial usage)" : ""}`],
      badges,
      reviewer: { state: m.turn?.role === "reviewer" ? "working" : "none", text: `${b?.verify?.profile || "Profile not recorded"}${m.turn?.role === "reviewer" ? ` · ${m.turn.short} active` : " · no active reviewer recorded"}` },
    });
  }
  return nodes;
}

// Exactly one prefix and recorded sender/recipient. This does not infer receipt.
export function exchangeFrames(entries, boundary, t, times, { animate = false, ownerChannel = false } = {}) {
  const available = entries.slice(0, Math.min(boundary, entries.length)).map((entry, i) => ({ entry, time: times[i] ?? 0 })).filter(({ entry }) => entry.link && (ownerChannel || (entry.link.from !== "@user" && entry.link.to !== "@user" && entry.link.channel !== "owner")));
  const latest = available.at(-1)?.time ?? 0;
  return available.filter(({ time }) => time >= (animate ? t : latest) - 2.6).slice(-14).map(({ entry, time }) => ({ entry, progress: animate ? Math.max(0, Math.min(1, (t - time) / 0.9)) : 1, past: (animate ? t : latest) - time > 1.5 }));
}

export function exchangeRoute(a, b, channel) {
  if (a.y === b.y) {
    const right = b.x > a.x, y = a.y + a.h / 2 + (right ? -8 : 8);
    return [{ x: right ? a.x + a.w : a.x, y }, { x: right ? b.x : b.x + b.w, y }];
  }
  const down = b.y > a.y;
  const x1 = a.x + a.w / 2 + (down ? 9 : -9), x2 = b.x + b.w / 2 + (down ? 9 : -9);
  const y1 = down ? a.y + a.h : a.y, y2 = down ? b.y : b.y + b.h;
  if (channel === "orchestrator") return [{ x: a.x + a.w, y: a.y + a.h / 2 }, { x: x2, y: a.y + a.h / 2 }, { x: x2, y: y2 }];
  const mid = (y1 + y2) / 2;
  return [{ x: x1, y: y1 }, { x: x1, y: mid }, { x: x2, y: mid }, { x: x2, y: y2 }];
}

export function pointAlong(points, progress) {
  const distances = points.slice(1).map((p, i) => Math.abs(p.x - points[i].x) + Math.abs(p.y - points[i].y));
  let remaining = distances.reduce((n, d) => n + d, 0) * Math.max(0, Math.min(1, progress));
  for (let i = 0; i < distances.length; i++) {
    const a = points[i], b = points[i+1], d = distances[i];
    if (!d) continue;
    if (remaining <= d || i === distances.length - 1) return { x: a.x + (b.x - a.x) * remaining / d, y: a.y + (b.y - a.y) * remaining / d, angle: Math.atan2(b.y - a.y, b.x - a.x) * 180 / Math.PI };
    remaining -= d;
  }
  return { ...points[0], angle: 0 };
}

// Short wording shared by the detailed inspector and its organization cards.
export function activity(m) {
  if (m.turn) return `${m.turn.role} turn ${m.turn.short}, ${m.turn.readOnly ? "read-only" : "writes"}`;
  if (m.checking) return "submission being checked";
  if (m.helpers?.length) return `${m.helpers.length} helpers active`;
  if (m.wait) return m.wait.outdated ? `wait outdated: ${m.wait.on?.length ? `needs ${m.wait.on.map(label).join(", ")}` : "current reason not recorded"}` : `waits: ${m.wait.text}`;
  if (m.state === "ready") return "ready for a turn";
  if (m.state === "done") return m.result ? `result ${m.result} accepted` : "done";
  if (!m.released && !m.state) return "not released in this Pass";
  return "";
}

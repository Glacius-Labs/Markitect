// Routing between the hub and the areas of the cockpit. The address keeps the
// place and its selection — a run with its pinned boundary, a Definition, a
// question or a proposal — so a view can be reopened. Escape leads from a
// run or a Definition to its area and from an area to the hub. The project
// list (#/projects) is where the owner opens, adds and sets up projects;
// while no project is open, every other place leads there.

import { show } from "./shell.js";
import { Inspector } from "./inspector.js";
import { renderHub } from "./hub.js";
import { renderModelView } from "./modelview.js";
import { renderSession } from "./session.js";
import { renderDecisions } from "./decisions.js";
import { renderRuns } from "./runs.js";
import { renderSettings } from "./settings.js";
import { renderProjects } from "./projects.js";
import { store } from "./store.js";
import { api } from "./api.js";

const app = document.getElementById("app");
let inspector = null;

export function parseRoute() {
  const hash = location.hash.replace(/^#/, "") || "/";
  const [path, query = ""] = hash.split("?");
  const params = new URLSearchParams(query);
  const parts = path.split("/").filter(Boolean).map(decodeURIComponent);
  switch (parts[0]) {
    case "run": return parts[1] ? { page: "run", run: parts[1], at: params.get("at"), sel: params.get("sel") } : { page: "runs" };
    case "runs": return { page: "runs" };
    case "model": return { page: "model", pkg: params.get("pkg"), id: params.get("def"), q: params.get("q") };
    case "session": return { page: "session" };
    case "decisions": case "proposals": return { page: "decisions" };
    case "proposal": return parts[1] ? { page: "decisions", item: { kind: "proposal", id: parts[1] } } : { page: "decisions" };
    case "settings": return { page: "settings" };
    case "projects": return { page: "projects", setup: params.get("setup") };
    default: return { page: "hub" };
  }
}

function modelHref(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const text = q.toString();
  return `#/model${text ? `?${text}` : ""}`;
}

function route() {
  const r = parseRoute();
  if (r.page !== "projects" && store.data.projects && !store.data.projects.current) {
    // The page may have been open before a project was: read the list again
    // and lead to it only when it still states that none is open.
    api.projects().then((view) => {
      store.data.projects = view;
      delete store.errors.projects;
      if (view.current) {
        store.refresh();
        route();
      } else {
        location.replace("#/projects");
      }
    }, () => location.replace("#/projects"));
    return;
  }
  if (r.page === "run" && inspector && inspector.run === r.run) {
    inspector.follow(r);
    return;
  }
  inspector?.destroy();
  inspector = null;
  switch (r.page) {
    case "run": {
      const content = show(app, { area: "runs", trail: [{ text: r.run }], escape: "#/runs", fill: true });
      inspector = new Inspector(content, r.run, r);
      break;
    }
    case "runs":
      renderRuns(show(app, { area: "runs" }));
      break;
    case "model": {
      const escape = r.id ? modelHref({ pkg: r.pkg, q: r.q }) : r.pkg ? modelHref({ q: r.q }) : null;
      const trail = r.id ? [{ text: `${r.pkg}/${r.id}` }] : r.pkg ? [{ text: r.pkg }] : [];
      renderModelView(show(app, { area: "model", tab: "browse", trail, escape, fill: true }), r);
      break;
    }
    case "session":
      renderSession(show(app, { area: "model", tab: "session", fill: true }));
      break;
    case "decisions": {
      const item = r.item || null;
      const trail = !item ? [] : [{ text: item.kind === "proposal" ? item.id : `${item.id} of ${item.run}` }];
      renderDecisions(show(app, { area: "decisions", trail, fill: true }), item);
      break;
    }
    case "settings":
      renderSettings(show(app, { area: "settings" }));
      break;
    case "projects":
      renderProjects(show(app, { page: "projects", trail: [{ text: "Projects" }], escape: store.data.projects?.current ? "#/" : null }), r.setup);
      break;
    default:
      renderHub(show(app, {}));
  }
}

window.addEventListener("hashchange", route);
// Which project is open decides where the first place is, so it is read first.
try {
  store.data.projects = await api.projects();
} catch {
  // The pages show what they cannot read.
}
route();

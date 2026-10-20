// Routing between the places of the cockpit. The address keeps the place and
// its selection — a run with its pinned boundary, a Definition, a proposal —
// so a view can be reopened.

import { show } from "./shell.js";
import { Inspector } from "./inspector.js";
import { renderOverview } from "./overview.js";
import { renderModelView } from "./modelview.js";
import { renderSession } from "./session.js";
import { renderProposals } from "./proposals.js";
import { renderRuns } from "./runs.js";
import { renderSettings } from "./settings.js";

const app = document.getElementById("app");
let inspector = null;

export function parseRoute() {
  const hash = location.hash.replace(/^#/, "") || "/";
  const [path, query = ""] = hash.split("?");
  const params = new URLSearchParams(query);
  const parts = path.split("/").filter(Boolean).map(decodeURIComponent);
  switch (parts[0]) {
    case "run": return { page: "run", run: parts[1], at: params.get("at"), sel: params.get("sel") };
    case "runs": return { page: "runs" };
    case "model": return { page: "model", pkg: params.get("pkg"), id: params.get("def"), q: params.get("q") };
    case "session": return { page: "session" };
    case "proposals": return { page: "proposals" };
    case "proposal": return { page: "proposals", id: parts[1] };
    case "settings": return { page: "settings" };
    default: return { page: "overview" };
  }
}

function route() {
  const r = parseRoute();
  if (r.page === "run" && inspector && inspector.run === r.run) {
    inspector.follow(r);
    return;
  }
  inspector?.destroy();
  inspector = null;
  switch (r.page) {
    case "run": {
      const content = show(app, { page: "runs", crumbs: [{ text: "Runs", href: "#/runs" }, { text: r.run }], fill: true });
      inspector = new Inspector(content, r.run, r);
      break;
    }
    case "runs":
      renderRuns(show(app, { page: "runs", crumbs: [{ text: "Runs" }] }));
      break;
    case "model":
      renderModelView(show(app, { page: "model", crumbs: [{ text: "Model", href: "#/model" }, ...(r.id ? [{ text: `${r.pkg}/${r.id}` }] : [])], fill: true }), r);
      break;
    case "session":
      renderSession(show(app, { page: "session", crumbs: [{ text: "Session" }], fill: true }));
      break;
    case "proposals":
      renderProposals(show(app, { page: "proposals", crumbs: [{ text: "Proposals", href: "#/proposals" }, ...(r.id ? [{ text: r.id }] : [])], fill: true }), r.id);
      break;
    case "settings":
      renderSettings(show(app, { page: "settings", crumbs: [{ text: "Settings" }] }));
      break;
    default:
      renderOverview(show(app, { page: "overview", crumbs: [{ text: "Overview" }] }));
  }
}

window.addEventListener("hashchange", route);
route();

// Routing between the cockpit entry, the run inspector, the execution settings,
// a proposal and the model session. The address keeps the run, the pinned boundary and the
// selection, so a view can be reopened.

import { renderRuns } from "./runs.js";
import { Inspector } from "./inspector.js";
import { renderSettings } from "./settings.js";
import { renderProposal } from "./proposal.js";
import { renderModel } from "./model.js";

const app = document.getElementById("app");
let current = null;

export function parseRoute() {
  const hash = location.hash.replace(/^#/, "") || "/";
  const [path, query = ""] = hash.split("?");
  const params = new URLSearchParams(query);
  const m = path.match(/^\/run\/([^/]+)$/);
  if (m) return { page: "run", run: decodeURIComponent(m[1]), at: params.get("at"), sel: params.get("sel") };
  if (path === "/settings") return { page: "settings" };
  if (path === "/model") return { page: "model" };
  const p = path.match(/^\/proposal\/([^/]+)$/);
  if (p) return { page: "proposal", id: decodeURIComponent(p[1]) };
  return { page: "runs" };
}

function route() {
  const r = parseRoute();
  if (r.page === "run") {
    if (current && current.run === r.run) {
      current.follow(r);
      return;
    }
    current?.destroy();
    current = new Inspector(app, r.run, r);
    return;
  }
  current?.destroy();
  current = null;
  if (r.page === "settings") renderSettings(app);
  else if (r.page === "proposal") renderProposal(app, r.id);
  else if (r.page === "model") renderModel(app);
  else renderRuns(app);
  window.scrollTo(0, 0);
}

window.addEventListener("hashchange", route);
route();

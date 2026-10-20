// Routing between the run list and the run inspector. The address keeps the
// run, the pinned boundary and the selection, so a view can be reopened.

import { renderRuns } from "./runs.js";
import { Inspector } from "./inspector.js";

const app = document.getElementById("app");
let current = null;

export function parseRoute() {
  const hash = location.hash.replace(/^#/, "") || "/";
  const [path, query = ""] = hash.split("?");
  const params = new URLSearchParams(query);
  const m = path.match(/^\/run\/([^/]+)$/);
  if (m) return { page: "run", run: decodeURIComponent(m[1]), at: params.get("at"), sel: params.get("sel") };
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
  renderRuns(app);
}

window.addEventListener("hashchange", route);
route();

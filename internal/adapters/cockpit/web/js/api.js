// The cockpit's interface. Every GET reads. answer writes an owner's answer
// through the owner channel the answer command uses, observeProject runs the
// repository's structural preparation again, proposeSettings makes a checked
// proposal without writing, and decide accepts (writes through Apply's
// guarded publication) or discards one. startSession and promptSession each
// start one provider turn of a model session; closeSession ends it.
// projects reads the per-user project list with the project that is open;
// addProject, openProject, removeProject, keepProject and setupProject change
// it (a setup also writes the project's runtime directory through the
// function behind markitect setup).
// runControl reads whether the cockpit can start a run and the run it
// started, with the confirmation facts when asked; startRun and stopRun start
// and stop that one run.

async function request(path, init = {}) {
  const res = await fetch(path, { ...init, headers: { Accept: "application/json", ...init.headers } });
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  return res.json();
}

const get = (path) => request(path);
const postJSON = (path, purpose, body) => request(path, {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": purpose },
  body: JSON.stringify(body),
});
const enc = encodeURIComponent;

export const api = {
  projects: () => get("/api/projects"),
  addProject: (body) => postJSON("/api/projects", "project-add", body),
  openProject: (name) => request(`/api/projects/${enc(name)}/open`, { method: "POST", headers: { "X-Markitect-Cockpit": "project-open" } }),
  removeProject: (name) => request(`/api/projects/${enc(name)}/remove`, { method: "POST", headers: { "X-Markitect-Cockpit": "project-remove" } }),
  keepProject: (name) => postJSON("/api/projects/keep", "project-keep", { name }),
  setupProject: (name, body) => postJSON(`/api/projects/${enc(name)}/setup`, "project-setup", body),
  project: () => get("/api/project"),
  model: () => get("/api/model"),
  observeProject: () => request("/api/project/observe", { method: "POST", headers: { "X-Markitect-Cockpit": "observe" } }),
  runs: () => get("/api/runs"),
  questions: () => get("/api/questions"),
  view: (run, at) => get(`/api/runs/${enc(run)}/view${at === undefined ? "" : `?at=${at}`}`),
  entries: (run, after = 0) => get(`/api/runs/${enc(run)}/entries?after=${after}`),
  detail: (run, ref, at) => get(`/api/runs/${enc(run)}/detail?ref=${enc(ref)}&at=${at}`),
  transcript: (run, turn, at) => get(`/api/runs/${enc(run)}/transcript?turn=${enc(turn)}&at=${at}`),
  settings: () => get("/api/settings"),
  proposeSettings: (settings) => request("/api/proposals/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "propose" },
    body: JSON.stringify(settings),
  }),
  proposals: () => get("/api/proposals"),
  proposal: (id) => get(`/api/proposals/${enc(id)}`),
  decide: (id, decision) => request(`/api/proposals/${enc(id)}/${decision}`, { method: "POST", headers: { "X-Markitect-Cockpit": decision } }),
  sessions: () => get("/api/sessions"),
  startSession: (prompt) => request("/api/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "start" },
    body: JSON.stringify({ prompt }),
  }),
  promptSession: (id, prompt) => request(`/api/sessions/${enc(id)}/prompt`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "prompt" },
    body: JSON.stringify({ prompt }),
  }),
  closeSession: (id) => request(`/api/sessions/${enc(id)}/close`, { method: "POST", headers: { "X-Markitect-Cockpit": "close" } }),
  runControl: (facts = false) => get(`/api/run-control${facts ? "?facts=1" : ""}`),
  // startRun names the committed head the confirmation was read from.
  startRun: (head) => request("/api/run-control/start", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "start-run" },
    body: JSON.stringify({ head }),
  }),
  stopRun: () => request("/api/run-control/stop", { method: "POST", headers: { "X-Markitect-Cockpit": "stop-run" } }),
  answer: (run, question, text) => request(`/api/runs/${enc(run)}/questions/${enc(question)}/answer`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "answer" },
    body: JSON.stringify({ text }),
  }),
};

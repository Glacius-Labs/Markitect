// The cockpit's interface. Every GET reads. answer writes an owner's answer
// through the owner channel the answer command uses, and observeProject runs
// the repository's structural preparation again; nothing here can start, stop
// or publish anything.

async function request(path, init = {}) {
  const res = await fetch(path, { ...init, headers: { Accept: "application/json", ...init.headers } });
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  return res.json();
}

const get = (path) => request(path);
const enc = encodeURIComponent;

export const api = {
  project: () => get("/api/project"),
  observeProject: () => request("/api/project/observe", { method: "POST", headers: { "X-Markitect-Cockpit": "observe" } }),
  runs: () => get("/api/runs"),
  questions: () => get("/api/questions"),
  view: (run, at) => get(`/api/runs/${enc(run)}/view${at === undefined ? "" : `?at=${at}`}`),
  entries: (run, after = 0) => get(`/api/runs/${enc(run)}/entries?after=${after}`),
  detail: (run, ref, at) => get(`/api/runs/${enc(run)}/detail?ref=${enc(ref)}&at=${at}`),
  transcript: (run, turn, at) => get(`/api/runs/${enc(run)}/transcript?turn=${enc(turn)}&at=${at}`),
  answer: (run, question, text) => request(`/api/runs/${enc(run)}/questions/${enc(question)}/answer`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Markitect-Cockpit": "answer" },
    body: JSON.stringify({ text }),
  }),
};

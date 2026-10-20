// The cockpit's read interface. Every call is a GET: nothing here can start,
// answer, stop or publish anything.

async function get(path) {
  const res = await fetch(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  return res.json();
}

const enc = encodeURIComponent;

export const api = {
  runs: () => get("/api/runs"),
  view: (run, at) => get(`/api/runs/${enc(run)}/view${at === undefined ? "" : `?at=${at}`}`),
  entries: (run, after = 0) => get(`/api/runs/${enc(run)}/entries?after=${after}`),
  detail: (run, ref, at) => get(`/api/runs/${enc(run)}/detail?ref=${enc(ref)}&at=${at}`),
  transcript: (run, turn, at) => get(`/api/runs/${enc(run)}/transcript?turn=${enc(turn)}&at=${at}`),
};

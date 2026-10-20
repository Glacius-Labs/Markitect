// The cockpit's shared observation: the project, the runs, the owner
// questions, the proposals and the model sessions, read together every few
// seconds while the page is visible. The desk rail, the status bar and the
// pages draw from it, so one poll serves them all. Every part keeps its last
// answer or its error; nothing here writes.

import { api } from "./api.js";

const POLL_MS = 4000;
const parts = {
  project: () => api.project(),
  runs: () => api.runs(),
  questions: () => api.questions(),
  proposals: () => api.proposals(),
  sessions: () => api.sessions(),
};

export const store = {
  data: {}, // part -> value
  errors: {}, // part -> Error
  listeners: new Set(),
  timer: null,
  inflight: null,

  subscribe(fn) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  },

  // refresh reads every part again and notifies once all arrived. Parts the
  // cockpit does not offer (proposals and sessions without a repository)
  // answer with an error that the pages show as "not configured".
  refresh() {
    if (this.inflight) return this.inflight;
    this.inflight = Promise.all(Object.entries(parts).map(([name, read]) => read().then(
      (value) => { this.data[name] = value; delete this.errors[name]; },
      (err) => { this.errors[name] = err; },
    ))).then(() => {
      this.inflight = null;
      for (const fn of this.listeners) {
        try {
          fn(this);
        } catch (err) {
          console.error(err);
        }
      }
    });
    return this.inflight;
  },

  start() {
    if (this.timer) return;
    this.refresh();
    this.timer = setInterval(() => {
      if (document.visibilityState === "visible") this.refresh();
    }, POLL_MS);
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible") this.refresh();
    });
  },

  // Derived views the desk and the pages share.
  get project() { return this.data.project?.project || null; },
  get openSession() { return (this.data.sessions?.sessions || []).find((s) => s.state !== "closed") || null; },
  get sessionsAvailable() { return Boolean(this.data.sessions?.available); },
  get openProposals() { return (this.data.proposals || []).filter((p) => p.state === "proposed" || p.state === "publishing"); },
  get liveRuns() { return (this.data.runs || []).filter((r) => r.health && (r.health.run === "running" || r.health.run === "finishing")); },
  get waitingQuestions() { return (this.data.questions || []).filter((q) => q.answerable); },
};

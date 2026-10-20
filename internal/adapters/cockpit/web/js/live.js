// The live link: records appended to a run's evidence and the current health
// of the run, over one server-sent event stream. Reconnecting resumes after
// the last record received, so nothing arrives twice and nothing is skipped.
// The link only reads; closing it or losing it never affects the run.

const STALE_MS = 6000;

export class LiveLink {
  constructor(run, after, handlers) {
    this.handlers = handlers;
    this.conn = "connecting";
    this.changed = Date.now();
    this.lastContact = 0;
    this.health = null;
    this.source = new EventSource(`/api/runs/${encodeURIComponent(run)}/stream?after=${after}`);
    this.source.onopen = () => this.set("connected");
    this.source.onerror = () => this.set(this.source.readyState === EventSource.CLOSED ? "closed" : "reconnecting");
    this.source.addEventListener("entries", (ev) => {
      this.contact();
      this.handlers.onEntries(JSON.parse(ev.data));
    });
    this.source.addEventListener("health", (ev) => {
      this.contact();
      this.health = JSON.parse(ev.data);
      this.handlers.onStatus();
    });
    this.source.addEventListener("replaced", (ev) => {
      this.contact();
      this.close("replaced");
      this.handlers.onReplaced(JSON.parse(ev.data));
    });
    this.timer = setInterval(() => this.handlers.onStatus(), 1000);
  }

  set(conn) {
    if (conn === this.conn) return;
    this.conn = conn;
    this.changed = Date.now();
    this.handlers.onStatus();
  }

  contact() {
    this.lastContact = Date.now();
    if (this.conn !== "connected") this.set("connected");
  }

  // stale says that a connected link has not heard from the server for longer
  // than its health interval allows.
  stale() {
    return this.conn === "connected" && this.lastContact > 0 && Date.now() - this.lastContact > STALE_MS;
  }

  close(reason = "closed") {
    clearInterval(this.timer);
    this.source.close();
    this.conn = reason;
    this.changed = Date.now();
  }
}

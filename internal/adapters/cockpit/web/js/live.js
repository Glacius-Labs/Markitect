// The live link: records appended to a run's evidence and the current health
// of the run, over one server-sent event stream. Reconnecting resumes after
// the last record received, so nothing arrives twice and nothing is skipped.
// The link only reads; closing it or losing it never affects the run.

const STALE_MS = 6000;

export class LiveLink {
  // after is the number of records held and mark the last one's mark, so the
  // server can tell whether it still has the same history.
  constructor(run, after, mark, handlers) {
    this.handlers = handlers;
    this.conn = "connecting";
    this.changed = Date.now();
    this.lastContact = 0;
    this.health = null;
    this.source = new EventSource(`/api/runs/${encodeURIComponent(run)}/stream?after=${after}&mark=${encodeURIComponent(mark || "")}`);
    this.source.onopen = () => this.set("connected");
    this.source.onerror = () => this.set(this.source.readyState === EventSource.CLOSED ? "closed" : "reconnecting");
    this.source.addEventListener("entries", (ev) => {
      this.contact();
      this.handlers.onEntries(JSON.parse(ev.data));
    });
    this.source.addEventListener("health", (ev) => {
      this.contact();
      const previous = this.health;
      this.health = JSON.parse(ev.data);
      this.handlers.onHealth?.(this.health, previous);
      this.handlers.onStatus();
      // A written report ends the run; nothing more will be recorded.
      if (this.health.run === "ended") this.close("ended");
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
    this.handlers.onConn?.(conn);
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

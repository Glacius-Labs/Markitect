// The run inspector: one recorded boundary at a time, with the organization,
// the account, the timeline and the exchanges of exactly that boundary.
// Playback moves the boundary through the record. Pausing it, selecting or
// reconnecting only changes what is shown; it never reaches the run. While a
// run is live the view follows new records until the user pins a boundary;
// it then counts the newer records and offers an explicit return to live.

import { api } from "./api.js";
import { h, clear, icon, clock, since, label, announce, when } from "./dom.js";
import { Playback, SPEEDS } from "./playback.js";
import { Diagram } from "./diagram.js";
import { Timeline } from "./timeline.js";
import { renderDetail } from "./detail.js";
import { LiveLink } from "./live.js";

const HIDDEN_BY_DEFAULT = new Set(["state", "runtime"]);

export class Inspector {
  constructor(app, run, route) {
    this.app = app;
    this.run = run;
    this.entries = [];
    this.views = new Map();
    this.boundary = 0;
    this.view = null;
    this.selection = route.sel || null;
    this.tab = "timeline";
    this.hiddenClasses = new Set(HIDDEN_BY_DEFAULT);
    this.followActivity = false;
    this.following = false;
    this.live = null;
    this.reduced = matchMedia("(prefers-reduced-motion: reduce)");
    this.motion = !this.reduced.matches;
    this.playback = new Playback((b, why) => this.onBoundary(b, why));
    this.onKey = (e) => this.key(e);
    this.onResize = () => this.renderTimeline();
    this.build();
    document.addEventListener("keydown", this.onKey);
    window.addEventListener("resize", this.onResize);
    this.start(route);
  }

  destroy() {
    this.dead = true;
    this.live?.close();
    this.playback.destroy();
    cancelAnimationFrame(this.anim);
    document.removeEventListener("keydown", this.onKey);
    window.removeEventListener("resize", this.onResize);
  }

  // follow applies an address change made outside the inspector.
  follow(route) {
    if (route.sel !== this.selection) this.select(route.sel || null, { fromRoute: true });
    if (route.at !== null && Number(route.at) !== this.boundary) this.playback.seek(Number(route.at));
  }

  async start(route) {
    try {
      const res = await api.entries(this.run, 0);
      if (this.dead) return;
      this.entries = res.entries;
    } catch (err) {
      clear(this.main);
      this.main.append(h("p", { role: "alert", class: "boot" }, `This run could not be read: ${err.message}`));
      return;
    }
    this.playback.setEntries(this.entries);
    this.renderLists();
    const at = route.at === null || route.at === undefined ? this.entries.length : Number(route.at);
    this.playback.seek(Number.isFinite(at) ? at : this.entries.length);
    this.following = this.boundary >= this.entries.length;
    this.live = new LiveLink(this.run, this.entries.length, this.entries.at(-1)?.mark, {
      onEntries: (p) => this.queueEntries(p),
      onHealth: (health, previous) => this.onHealth(health, previous),
      onStatus: () => this.renderLive(),
      onConn: (conn) => announce({ connected: "Live link connected.", reconnecting: "Live link lost; reconnecting." }[conn] || ""),
      onReplaced: () => this.onReplaced(),
    });
  }

  // ---------- live ----------

  isLive() {
    const run = this.live?.health?.run;
    return run === "running" || run === "finishing";
  }

  // queueEntries applies deliveries one at a time, so a refill cannot overlap
  // another delivery.
  queueEntries(p) {
    this.pending = (this.pending || Promise.resolve()).then(() => this.onEntries(p)).catch((err) => {
      // A refill beyond the record means the history changed under the view.
      if (/beyond the record/.test(err.message)) this.onReplaced();
    });
  }

  // onEntries appends records the live link delivered. A delivery that
  // overlaps what is held is trimmed; one that would leave a gap is refilled
  // from the read interface, so the held record stays one complete prefix.
  async onEntries(p) {
    let delivered = p;
    if (p.from > this.entries.length + 1) {
      const res = await api.entries(this.run, this.entries.length);
      delivered = { from: this.entries.length + 1, entries: res.entries };
    }
    const items = delivered.entries.slice(this.entries.length - delivered.from + 1);
    if (!items.length || this.dead) return;
    this.entries = this.entries.concat(items);
    this.playback.setEntries(this.entries);
    this.appendRows(items);
    this.range.max = String(this.entries.length);
    if (this.following) this.playback.catchUp();
    this.renderPosition();
    this.renderLive();
  }

  // onHealth reacts to what the record cannot announce itself: a report is
  // written after the last record, so the final view is read again then.
  onHealth(health, previous) {
    const key = (x) => x && `${x.run}/${x.source.report}/${x.source.replaced}`;
    if (key(health) !== key(previous)) {
      if (previous) announce(`Run state: ${health.run.replaceAll("-", " ")}.`);
      if (previous && this.boundary >= this.entries.length) {
        this.refetch = true;
        this.want = this.boundary;
        this.fetchView();
      }
    }
  }

  onReplaced() {
    if (this.replaced) return;
    this.replaced = true;
    this.live?.close("replaced");
    this.following = false;
    this.playback.pause("pause");
    this.accountEl.prepend(h("p", { role: "alert", class: "st-fail" }, "The events file was replaced while it was observed. The view keeps what it read; reload the page to read the new file from its start."));
    this.renderLive();
  }

  returnToLive() {
    this.following = true;
    if (this.playback.playing) this.playback.bound();
    else if (this.boundary < this.entries.length) this.playback.catchUp();
    this.renderLive();
    announce("Following the run again.");
  }

  renderLive() {
    if (!this.liveEl) return;
    const l = this.live;
    const health = l?.health;
    const live = this.isLive();
    const newer = this.entries.length - this.boundary;
    const secs = (t) => Math.max(0, Math.round((Date.now() - t) / 1000));
    let conn;
    switch (l?.conn) {
      case "connected": conn = l.stale() ? `no contact for ${secs(l.lastContact)} s; the shown data may be stale` : `connected, last contact ${secs(l.lastContact)} s ago`; break;
      case "reconnecting": conn = `connection lost, reconnecting for ${secs(l.changed)} s`; break;
      case "replaced": conn = "stopped: the events file was replaced"; break;
      case "ended": conn = "closed: the run's report is written"; break;
      case "connecting": conn = "connecting"; break;
      default: conn = "not connected";
    }
    const runText = {
      running: "Run is running", finishing: "Run is finishing: outcome recorded, publication and report pending",
      ended: "Run ended; its report is written", "ended-without-report": "Run ended without a report", unknown: "Run state unknown",
    }[health?.run] || "Run state not observed yet";
    const current = l?.conn === "connected" && !l.stale();
    const cls = live ? (current ? "st-done" : "st-waiting") : "st-none";
    const runSaid = current || !health ? runText : `At the last contact: ${runText.charAt(0).toLowerCase()}${runText.slice(1)}`;
    const follow = live ? ` ${this.following ? "Following new records." : `Pinned at record ${this.boundary}${newer ? `, ${newer} newer` : ""}.`}` : "";
    clear(this.liveEl).append(
      h("span", { class: `outcome ${cls}` }, h("span", { class: "lamp" }), h("strong", {}, live ? "Live" : "Replay")),
      h("span", { class: "muted", title: health?.why || "" }, `${runSaid}.${follow} Link ${conn}.`));
    this.liveBtn.hidden = !live || this.following;
    this.liveBtn.textContent = newer ? `Return to live (${newer} newer)` : "Return to live";
  }

  // ---------- structure ----------

  build() {
    clear(this.app);
    document.title = `${this.run} – Markitect cockpit`;
    this.headEl = h("header", { class: "head" });
    this.liveEl = h("span", { class: "mode", "aria-live": "off", title: "The current observation of the run, separate from the recorded boundary. Pausing or losing the link never affects the run." });
    this.railEl = h("nav", { class: "rail", "aria-label": "Passes" });
    this.accountEl = h("section", { class: "account", "aria-label": "Account at this boundary", "aria-live": "polite" });
    this.stageEl = h("div", { class: "stage" });
    this.legend = h("div", { class: "legend", "aria-hidden": "true" },
      h("span", {}, h("i", { class: "l-agent" }), "Agent message"),
      h("span", {}, h("i", { class: "l-submission" }), "Submission"),
      h("span", {}, h("i", { class: "l-orchestrator" }), "Orchestrator control"),
      h("span", {}, h("i", { class: "l-owner" }), "Owner channel"));
    this.stageEl.append(this.legend);
    this.diagram = new Diagram(this.stageEl, { onSelect: (ref) => this.select(ref) });
    this.diagram.setMotion(this.motion);

    this.tabButtons = {};
    const tab = (id, text) => (this.tabButtons[id] = h("button", { role: "tab", id: `tab-${id}`, "aria-controls": `panel-${id}`, "aria-selected": String(this.tab === id), onclick: () => this.setTab(id) }, text));
    this.zoomOut = h("button", { class: "btn icon", "aria-label": "Zoom the timeline out", onclick: () => this.timeline.setZoom(this.timeline.zoom / 2) }, "−");
    this.zoomIn = h("button", { class: "btn icon", "aria-label": "Zoom the timeline in", onclick: () => this.timeline.setZoom(this.timeline.zoom * 2) }, "+");
    const tabs = h("div", { class: "tabs", role: "tablist" }, tab("timeline", "Timeline"), tab("exchanges", "Exchanges"), tab("events", "Events"),
      h("div", { class: "tools" }, this.zoomOut, this.zoomIn));
    this.panels = {
      timeline: h("div", { class: "tabpanel timeline", id: "panel-timeline", role: "tabpanel", "aria-labelledby": "tab-timeline" }),
      exchanges: h("div", { class: "tabpanel", id: "panel-exchanges", role: "tabpanel", "aria-labelledby": "tab-exchanges", hidden: true }),
      events: h("div", { class: "tabpanel", id: "panel-events", role: "tabpanel", "aria-labelledby": "tab-events", hidden: true }),
    };
    this.timeline = new Timeline(this.panels.timeline, { onSelect: (ref) => this.select(ref) });
    const lower = h("section", { class: "lower", "aria-label": "Timeline and record" }, tabs, h("div", { style: { minHeight: 0, display: "grid" } }, Object.values(this.panels)));
    this.main = h("main", { id: "main", class: "work" }, this.stageEl, lower);
    this.asideEl = h("aside", { class: "aside", "aria-label": "Details" });
    this.transportEl = h("footer", { class: "transport", "aria-label": "Replay" });
    this.root = h("div", { class: "inspector" }, this.headEl, this.railEl, this.accountEl, this.main, this.asideEl, this.transportEl);
    this.app.append(this.root);
    this.buildTransport();
  }

  buildTransport() {
    const pb = this.playback;
    this.playBtn = h("button", { class: "btn primary", "aria-label": "Play the replay", title: "Play the replay (Space)", onclick: () => this.togglePlay() }, icon("play"));
    const btn = (name, text, fn) => h("button", { class: "btn icon", "aria-label": text, title: text, onclick: fn }, icon(name));
    this.range = h("input", { type: "range", min: 0, max: 0, value: 0, step: 1, "aria-label": "Boundary in the record" });
    this.range.addEventListener("input", () => {
      pb.pause("scrub");
      pb.seek(Number(this.range.value));
    });
    this.marks = h("div", { class: "marks" });
    this.speedSel = h("select", { "aria-label": "Replay speed", onchange: () => pb.setSpeed(Number(this.speedSel.value)) },
      SPEEDS.map((sp) => h("option", { value: sp, selected: sp === 1 }, `${sp}×`)));
    this.posEl = h("div", { class: "pos", "aria-live": "off" });
    this.followBtn = h("button", { class: "btn", "aria-pressed": "false", title: "Select the Manager that acts at each step while playing", onclick: () => {
      this.followActivity = !this.followActivity;
      this.followBtn.setAttribute("aria-pressed", String(this.followActivity));
    } }, "Follow activity");
    this.motionBtn = h("button", { class: "btn", "aria-pressed": String(this.motion), title: "Animate exchanges as moving arrows", onclick: () => {
      this.motion = !this.motion;
      this.motionBtn.setAttribute("aria-pressed", String(this.motion));
      this.diagram.setMotion(this.motion);
      this.drawExchanges();
    } }, "Motion");
    this.keysBtn = h("button", { class: "btn icon", "aria-label": "Keyboard shortcuts", "aria-expanded": "false", onclick: () => this.toggleKeys() }, "?");
    this.liveBtn = h("button", { class: "btn", hidden: true, title: "Show the latest records and follow the run again", onclick: () => this.returnToLive() }, "Return to live");
    this.transportEl.append(
      h("div", { class: "group" },
        btn("start", "Go to the start (Home)", () => { pb.pause("seek"); pb.seek(0); }),
        btn("stepb", "Previous record (Left)", () => this.step(-1)),
        this.playBtn,
        btn("stepf", "Next record (Right)", () => this.step(1)),
        btn("end", "Go to the end (End)", () => { pb.pause("seek"); pb.seek(this.entries.length); })),
      this.speedSel,
      h("div", { class: "scrub" }, this.marks, this.range),
      this.posEl,
      this.liveBtn,
      h("div", { class: "group" }, this.followBtn, this.motionBtn, this.keysBtn));
  }

  // ---------- boundary and views ----------

  onBoundary(b, why) {
    this.boundary = b;
    // A boundary the user chooses pins the view; reaching the latest record
    // by choice or by playing on resumes following a live run.
    if (why === "pause" || why === "select") this.following = false;
    else if (why === "seek" || why === "scrub" || why === "end" || why === "caught-up") this.following = b >= this.entries.length;
    this.range.max = String(this.entries.length);
    this.range.value = String(b);
    this.updatePlayButton();
    this.renderPosition();
    if (why !== "play") this.writeAddress();
    this.want = b;
    this.fetchView();
    if (this.playback.playing) this.startAnimation();
    else this.drawExchanges();
    if (why === "end") announce(this.isLive() ? "Replay reached the latest record; following the run." : "Replay reached the end of the record.");
    this.renderLive();
  }

  async fetchView() {
    if (this.fetching) return;
    while (!this.dead && this.want !== undefined && (this.refetch || this.view?.boundary.seq !== this.want)) {
      this.refetch = false;
      const b = this.want;
      let v = this.views.get(b);
      if (!v) {
        this.fetching = true;
        try {
          v = await api.view(this.run, b);
        } catch (err) {
          this.fetching = false;
          this.accountEl.replaceChildren(h("p", { role: "alert" }, `The view at record ${b} could not be read: ${err.message}`));
          return;
        }
        this.fetching = false;
        // A final view changes when records or the report arrive; only earlier
        // boundaries are fixed and cached.
        if (!v.boundary.final) {
          if (this.views.size > 300) this.views.delete(this.views.keys().next().value);
          this.views.set(b, v);
        }
      }
      if (this.want === b) this.show(v);
    }
  }

  show(v) {
    this.view = v;
    if (this.followActivity && this.playback.playing) {
      const actor = this.latestActor(v.boundary.seq);
      if (actor && this.selection !== `manager:${actor}`) this.selection = `manager:${actor}`;
    }
    this.renderHead();
    this.renderRail();
    this.renderAccount();
    this.diagram.update(v, this.entries, this.boundary, this.playback.times, this.selection);
    this.renderTimeline();
    this.markLists();
    this.renderMarks();
    const now = performance.now();
    if (!this.playback.playing || !this.lastDetail || now - this.lastDetail > 700) {
      this.lastDetail = now;
      this.renderAside();
    }
  }

  // latestActor is the Manager that acted in the latest work or exchange
  // record; Orchestrator and owner actions select nobody.
  latestActor(seq) {
    const managers = new Set(this.view?.managers.map((m) => m.path));
    for (let i = seq - 1; i >= 0; i--) {
      const e = this.entries[i];
      if (e.class !== "work" && e.class !== "exchange") continue;
      const actor = e.link ? e.link.from : e.manager;
      if (managers.has(actor)) return actor;
    }
    return null;
  }

  // ---------- rendering ----------

  renderHead() {
    const v = this.view;
    const st = v.status;
    const rep = st.report;
    const outcome = rep ? rep.outcome : st.outcome;
    const cls = { success: "st-done", verified: "st-done", failed: "st-fail", incomplete: "st-waiting" }[outcome] || (outcome ? "" : "st-working");
    const text = rep ? `Run outcome: ${rep.outcome}` : st.outcome ? `Outcome recorded: ${st.outcome}` : v.boundary.final ? "No outcome recorded" : "In progress at this boundary";
    clear(this.headEl).append(
      h("a", { class: "home", href: "#/" }, "Markitect cockpit"),
      h("span", { class: "runid" }, v.run.id),
      h("span", { class: "runmeta" }, `${v.managers.length} Manager${v.managers.length === 1 ? "" : "s"}, ${v.run.started ? `started ${when(v.run.started)}` : "not started"}`),
      h("span", { class: "spacer" }),
      h("span", { class: `outcome ${cls}`, title: "What the record shows at this boundary" }, h("span", { class: "lamp" }), text),
      this.liveEl);
    this.renderLive();
  }

  renderRail() {
    const v = this.view;
    const items = v.passes.filter((p) => p.started).map((p) => {
      const stateCls = p.sealed ? `sealed-${p.verdict}` : v.status.outcome ? "stopped" : "running";
      const meta = p.sealed
        ? `${p.verdict === "satisfied" ? "sealed satisfied" : "needs correction"}${p.review && p.checks?.length ? `, checks ${p.checks.every((c) => c.passed) ? "passed" : "failed"}` : ""}`
        : v.status.outcome ? "ended unsealed" : "running";
      const extra = p.correction ? `correction ${p.addresses.join(", ")}` : p.review && p.candidate ? `candidate ${p.candidate.slice(0, 8)}` : p.writes ? "owned writes" : "read-only";
      return h("button", { class: `pass ${stateCls}${this.selection === `pass:${p.id}` ? " selected" : ""}`, onclick: () => this.select(`pass:${p.id}`), "aria-label": `${p.purpose} Pass ${p.id}: ${meta}` },
        h("span", { class: "pname" }, `${cap(p.purpose)} ${p.id}`), h("span", { class: "pmeta" }, meta), h("span", { class: "pmeta" }, extra));
    });
    const rep = v.status.report;
    if (rep) {
      const pub = rep.publication;
      const ok = rep.outcome === "success";
      items.push(h("button", { class: `pass publish ${ok ? "ok" : "bad"}${this.selection === "run" ? " selected" : ""}`, onclick: () => this.select("run") },
        h("span", { class: "pname" }, "Publish"),
        h("span", { class: "pmeta" }, pub ? `${pub.outcome}, ${pub.paths.filter((p) => p.state === "after").length}/${pub.paths.length} read back` : "nothing published"),
        h("span", { class: "pmeta" }, `run ${rep.outcome}`)));
    }
    if (!items.length) items.push(h("span", { class: "muted" }, "No Pass has started at this boundary."));
    clear(this.railEl).append(...items);
  }

  renderAccount() {
    clear(this.accountEl);
    const v = this.view;
    if (this.live?.health?.run === "ended-without-report" && v.boundary.final && !v.status.outcome) {
      this.accountEl.append(h("p", { role: "alert", class: "st-fail" }, "The run is no longer running and its record ends without an outcome or report. What it did after the last record is unknown; work shown as running was running when the record ends."));
    }
    for (const sentence of this.view.account) {
      this.accountEl.append(h("p", {}, sentence.map((seg) => seg.ref ? h("button", { class: "ref", onclick: () => this.select(seg.ref) }, seg.text) : seg.text)));
    }
  }

  renderTimeline() {
    if (!this.view || this.tab !== "timeline") return;
    const first = this.entries.find((e) => e.at)?.at;
    const last = [...this.entries].reverse().find((e) => e.at)?.at;
    this.timeline.render(this.view, this.entries, this.boundary, { start: this.view.run.started || first, end: last || first }, this.selection);
  }

  listRow(e) {
    const li = h("li", { class: `c-${e.class}`, dataset: { seq: e.seq }, tabindex: -1 },
      h("span", { class: "t" }, `${clock(e.at)}`),
      h("span", { class: "x" }, e.link ? h("span", { class: `chan ch-${e.link.channel}` }, exchangeKind(e)) : null, e.text));
    li.addEventListener("click", () => {
      this.playback.pause("select");
      this.playback.seek(e.seq);
      this.select(e.ref, { keepPlaying: true });
    });
    return li;
  }

  appendRows(items) {
    this.eventsList.append(...items.map((e) => this.listRow(e)));
    this.exchangeList.append(...items.filter((e) => e.link || e.kind === "question.lapsed").map((e) => this.listRow(e)));
    this.markLists();
  }

  renderLists() {
    const filters = h("div", { class: "filters", role: "group", "aria-label": "Shown record classes" },
      ["exchange", "work", "control", "evidence", "problem", "state", "runtime"].map((cls) => {
        const box = h("input", { type: "checkbox", checked: !this.hiddenClasses.has(cls) });
        box.addEventListener("change", () => {
          if (box.checked) this.hiddenClasses.delete(cls);
          else this.hiddenClasses.add(cls);
          this.markLists();
        });
        return h("label", {}, box, cls);
      }));
    this.eventsList = h("ul", { class: "list", "aria-label": "Records up to the boundary" }, this.entries.map((e) => this.listRow(e)));
    this.exchangeList = h("ul", { class: "list", "aria-label": "Exchanges up to the boundary" }, this.entries.filter((e) => e.link || e.kind === "question.lapsed").map((e) => this.listRow(e)));
    clear(this.panels.events).append(filters, this.eventsList);
    clear(this.panels.exchanges).append(
      h("p", { class: "note", style: { padding: "6px 12px", margin: 0 } }, "Every exchange is a recorded send, answer or decision. Delivery into a turn and reading are not recorded."),
      this.exchangeList);
  }

  markLists() {
    const b = this.boundary;
    for (const list of [this.eventsList, this.exchangeList]) {
      if (!list) continue;
      let current = null;
      for (const li of list.children) {
        const seq = Number(li.dataset.seq);
        const cls = [...li.classList].find((c) => c.startsWith("c-"))?.slice(2);
        li.classList.toggle("later", seq > b || (list === this.eventsList && this.hiddenClasses.has(cls)));
        li.classList.remove("current");
        if (seq <= b && !li.classList.contains("later")) current = li;
      }
      if (current) {
        current.classList.add("current");
        if (!list.hidden && list.offsetParent) current.scrollIntoView({ block: "nearest" });
      }
    }
  }

  renderMarks() {
    clear(this.marks);
    const total = Math.max(1, this.entries.length);
    for (const p of this.view.passes) {
      if (!p.startSeq) continue;
      const end = p.sealedSeq || this.boundary;
      const color = p.sealed ? (p.verdict === "satisfied" ? "var(--done)" : "var(--fail)") : "var(--work)";
      this.marks.append(h("span", { style: { position: "absolute", left: `${(p.startSeq / total) * 100}%`, width: `${Math.max(0.3, ((end - p.startSeq) / total) * 100)}%`, top: "5px", height: "6px", background: color, opacity: 0.55 } }));
    }
    this.marks.append(h("span", { style: { position: "absolute", left: 0, right: 0, top: "7px", height: "2px", background: "var(--rule)", zIndex: -1 } }));
  }

  renderPosition() {
    const e = this.entries[this.boundary - 1];
    const startAt = this.entries[0]?.at;
    clear(this.posEl).append(
      h("strong", {}, `Record ${this.boundary} of ${this.entries.length}`),
      e ? ` at ${clock(e.at)}, ${since(startAt, e.at)} into the run` : " (before the first record)");
  }

  renderAside() {
    renderDetail(this.asideEl, {
      view: this.view, entries: this.entries, boundary: this.boundary, ref: this.selection,
      select: (ref) => this.select(ref),
      detail: (ref) => api.detail(this.run, ref, this.boundary),
      openTranscript: (turn) => this.openTranscript(turn),
    });
    this.asideEl.classList.toggle("open", Boolean(this.selection));
  }

  // ---------- interaction ----------

  select(ref, opts = {}) {
    // A manual selection pins the presentation: replay pauses, the run does not.
    if (this.playback.playing && !opts.keepPlaying) {
      this.playback.pause("select");
      announce("Replay paused for the selection. The run is not affected.");
    }
    this.selection = ref;
    if (!opts.fromRoute) this.writeAddress();
    if (this.view) {
      this.renderRail();
      this.diagram.update(this.view, this.entries, this.boundary, this.playback.times, this.selection);
      this.renderTimeline();
      this.lastDetail = performance.now();
      this.renderAside();
    }
  }

  setTab(id) {
    this.tab = id;
    for (const [k, b] of Object.entries(this.tabButtons)) b.setAttribute("aria-selected", String(k === id));
    for (const [k, p] of Object.entries(this.panels)) p.hidden = k !== id;
    this.zoomIn.hidden = this.zoomOut.hidden = id !== "timeline";
    if (id === "timeline") this.renderTimeline();
    else this.markLists();
  }

  togglePlay() {
    if (this.playback.playing) this.playback.pause("pause");
    else if (this.isLive() && this.boundary >= this.entries.length) this.returnToLive();
    else if (this.isLive()) this.playback.resume();
    else this.playback.play();
    this.updatePlayButton();
  }

  updatePlayButton() {
    const playing = this.playback.playing;
    this.playBtn.replaceChildren(icon(playing ? "pause" : "play"));
    this.playBtn.setAttribute("aria-label", playing ? "Pause the replay (the run is not affected)" : "Play the replay");
    this.playBtn.title = playing ? "Pause the replay (Space). Pausing never affects the run." : "Play the replay (Space)";
  }

  step(dir, exchangesOnly = false) {
    this.playback.pause("seek");
    let b = this.boundary + dir;
    if (exchangesOnly) {
      while (b > 0 && b <= this.entries.length && !this.entries[b - 1].link) b += dir;
    } else {
      // Skip bookkeeping records hidden from the event list.
      while (b > 0 && b <= this.entries.length && this.hiddenClasses.has(this.entries[b - 1].class)) b += dir;
    }
    this.playback.seek(Math.max(0, Math.min(this.entries.length, b)));
  }

  startAnimation() {
    if (this.animating) return;
    this.animating = true;
    const frame = () => {
      if (this.dead || !this.playback.playing) {
        this.animating = false;
        this.drawExchanges();
        return;
      }
      this.diagram.boundary = this.boundary;
      this.diagram.drawExchanges(this.playback.t, true);
      this.anim = requestAnimationFrame(frame);
    };
    this.anim = requestAnimationFrame(frame);
  }

  drawExchanges() {
    if (!this.view) return;
    this.diagram.boundary = this.boundary;
    this.diagram.drawExchanges(this.playback.t, false);
  }

  writeAddress() {
    const q = new URLSearchParams();
    q.set("at", String(this.boundary));
    if (this.selection) q.set("sel", this.selection);
    history.replaceState(null, "", `#/run/${encodeURIComponent(this.run)}?${q}`);
  }

  toggleKeys() {
    if (this.keysEl) {
      this.keysEl.remove();
      this.keysEl = null;
      this.keysBtn.setAttribute("aria-expanded", "false");
      return;
    }
    const k = (key, text) => [h("dt", {}, h("kbd", {}, key)), h("dd", {}, text)];
    this.keysEl = h("div", { class: "keys", role: "dialog", "aria-label": "Keyboard shortcuts" }, h("dl", {},
      k("Space", "Play or pause the replay"), k("← →", "Previous or next record"), k("Shift + ← →", "Previous or next exchange"),
      k("Home End", "Start or end of the record"), k("[ ]", "Slower or faster"), k("Esc", "Clear the selection or close"), k("?", "Show or hide this list")),
      h("p", { class: "note" }, "Pausing the replay never affects a run."));
    this.app.append(this.keysEl);
    this.keysBtn.setAttribute("aria-expanded", "true");
  }

  key(e) {
    if (e.target.closest("input, select, textarea") && e.key !== "Escape") return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    const pb = this.playback;
    switch (e.key) {
      case " ":
        if (e.target.closest("button, [role=button]")) return;
        e.preventDefault();
        this.togglePlay();
        break;
      case "ArrowRight": e.preventDefault(); this.step(1, e.shiftKey); break;
      case "ArrowLeft": e.preventDefault(); this.step(-1, e.shiftKey); break;
      case "Home": e.preventDefault(); pb.pause("seek"); pb.seek(0); break;
      case "End": e.preventDefault(); pb.pause("seek"); pb.seek(this.entries.length); break;
      case "[": case "]": {
        const i = SPEEDS.indexOf(pb.speed) + (e.key === "]" ? 1 : -1);
        if (i >= 0 && i < SPEEDS.length) {
          pb.setSpeed(SPEEDS[i]);
          this.speedSel.value = String(SPEEDS[i]);
          announce(`Replay speed ${SPEEDS[i]}×`);
        }
        break;
      }
      case "?": this.toggleKeys(); break;
      case "Escape":
        if (this.transcriptEl) this.closeTranscript();
        else if (this.keysEl) this.toggleKeys();
        else if (this.selection) this.select(null);
        break;
    }
  }

  // ---------- transcript ----------

  async openTranscript(turn) {
    this.playback.pause("select");
    this.closeTranscript();
    const body = h("div", { class: "body" }, h("p", { class: "note" }, "Reading the transcript…"));
    const closeBtn = h("button", { class: "btn", onclick: () => this.closeTranscript() }, "Close");
    this.scrimEl = h("div", { class: "scrim", onclick: () => this.closeTranscript() });
    this.transcriptEl = h("section", { class: "transcript", role: "dialog", "aria-modal": "true", "aria-label": `Transcript of turn ${turn}` },
      h("header", {}, h("h2", {}, `Transcript of ${turn.slice(turn.lastIndexOf("-") + 1)}`), h("span", { class: "muted" }, turn), h("span", { class: "spacer", style: { flex: 1 } }), closeBtn), body);
    this.app.append(this.scrimEl, this.transcriptEl);
    closeBtn.focus();
    try {
      const t = await api.transcript(this.run, turn, this.boundary);
      clear(body);
      if (t.until) body.append(h("p", { class: "note" }, `Cut at the boundary's recorded time ${clock(t.until)}; ${t.later} later line${t.later === 1 ? "" : "s"} not shown.`));
      if (t.truncated) body.append(h("p", { class: "note" }, "Long entries are shortened here; the file keeps them whole."));
      if (t.unread) body.append(h("p", { class: "note st-fail" }, `${t.unread} line${t.unread === 1 ? "" : "s"} could not be read.`));
      body.append(h("p", { class: "note" }, "The provider stream as the adapter recorded it, with secrets redacted by the producer."));
      for (const e of t.entries) {
        body.append(h("div", { class: `tline f-${e.from}` },
          h("div", { class: "who" }, whoText(e), h("div", { class: "faint" }, clock(e.at))),
          h("div", { class: `txt ${e.from === "assistant" ? "prose" : ""}` }, e.text)));
      }
    } catch (err) {
      clear(body).append(h("p", { role: "alert" }, `The transcript could not be read: ${err.message}`));
    }
  }

  closeTranscript() {
    this.transcriptEl?.remove();
    this.scrimEl?.remove();
    this.transcriptEl = this.scrimEl = null;
  }
}

function whoText(e) {
  switch (e.from) {
    case "input": return "Input";
    case "assistant": return "Agent";
    case "thinking": return "Agent thinking";
    case "tool": return `Tool ${e.name || ""}`;
    case "result": return "Tool result";
    case "orchestrator": return "Orchestrator";
    case "runtime": return "Runtime";
    default: return e.name ? `Provider ${e.name}` : "Provider";
  }
}

// exchangeKind names what an exchange is; the channel class tells who
// authored it (an Agent, the Orchestrator or the owner channel).
function exchangeKind(e) {
  if (e.kind === "proposal.submitted") return "submission";
  if (e.kind === "manager.release") return "release";
  if (e.kind === "question.asked") return e.link.channel === "owner" ? "owner question" : "question";
  if (e.kind === "question.answered") return e.link.channel === "owner" ? "owner answer" : "answer";
  if (e.kind === "message") return "information";
  return "decision";
}

function cap(s) {
  return s ? s[0].toUpperCase() + s.slice(1) : "";
}

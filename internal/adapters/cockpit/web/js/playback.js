// Presentation playback over recorded entries. It moves a boundary through
// the record; it never touches the run. Recorded gaps are compressed so quiet
// work does not stall the replay, and bursts of bookkeeping records take no
// time of their own.

export const SPEEDS = [0.5, 1, 2, 4, 8, 16];
export const TOKEN_SECONDS = 0.9;

const COMPRESS = 8; // recorded seconds per playback second at 1x
const MAX_GAP = 1.6; // playback seconds a quiet period takes at most
const MIN_STEP = 0.35; // playback seconds a visible step takes at least

const quiet = new Set(["state", "runtime"]);

export class Playback {
  constructor(onBoundary) {
    this.onBoundary = onBoundary;
    this.entries = [];
    this.times = [];
    this.t = 0;
    this.boundary = 0;
    this.speed = 1;
    this.playing = false;
    this.frame = 0;
    this.last = 0;
  }

  // setEntries takes the record's entries in order; it only appends times for
  // new entries, so a growing record keeps earlier positions.
  setEntries(entries) {
    for (let i = this.times.length; i < entries.length; i++) {
      const e = entries[i];
      if (i === 0) {
        this.times.push(0);
        continue;
      }
      const gap = Math.max(0, (Date.parse(e.at) - Date.parse(entries[i - 1].at)) / 1000) / COMPRESS;
      const min = quiet.has(e.class) ? 0 : MIN_STEP;
      this.times.push(this.times[i - 1] + Math.min(MAX_GAP, Math.max(min, gap)));
    }
    this.entries = entries;
  }

  // seek places the boundary after record seq and the clock at its time.
  seek(seq) {
    seq = Math.max(0, Math.min(seq, this.entries.length));
    this.boundary = seq;
    this.t = seq > 0 ? this.times[seq - 1] : 0;
    this.onBoundary(seq, "seek");
  }

  play() {
    if (this.playing) return;
    if (this.boundary >= this.entries.length) this.seek(0);
    this.playing = true;
    this.last = performance.now();
    const step = (now) => {
      if (!this.playing) return;
      const dt = Math.min(0.25, (now - this.last) / 1000);
      this.last = now;
      this.t += dt * this.speed;
      let b = this.boundary;
      while (b < this.entries.length && this.times[b] <= this.t) b++;
      if (b !== this.boundary) {
        this.boundary = b;
        this.onBoundary(b, "play");
      }
      if (this.boundary >= this.entries.length && this.t > this.times[this.times.length - 1] + TOKEN_SECONDS) {
        this.pause("end");
        return;
      }
      this.frame = requestAnimationFrame(step);
    };
    this.frame = requestAnimationFrame(step);
  }

  pause(reason = "pause") {
    if (!this.playing) return;
    this.playing = false;
    cancelAnimationFrame(this.frame);
    this.onBoundary(this.boundary, reason);
  }

  setSpeed(speed) {
    this.speed = speed;
  }

  destroy() {
    this.playing = false;
    cancelAnimationFrame(this.frame);
  }
}

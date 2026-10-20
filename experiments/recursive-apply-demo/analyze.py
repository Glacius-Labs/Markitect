"""Summarize one Apply from its evidence and report. Usage: analyze.py STATE_DIR APPLY_JSON"""
import json
import sys
from datetime import datetime
from pathlib import Path

state, report_path = Path(sys.argv[1]), Path(sys.argv[2])
runs = sorted((state / "runs").iterdir())
run = runs[-1]
events = [json.loads(line) for line in (run / "events.jsonl").read_text().splitlines()]
report = json.loads(report_path.read_text()) if report_path.exists() and report_path.stat().st_size else {}


def at(e):
    return datetime.fromisoformat(e["at"].replace("Z", "+00:00"))


t0 = at(events[0])
rel = lambda e: round((at(e) - t0).total_seconds(), 1)

turns, open_turns, peak, busy = {}, {}, 0, {"execute": 0.0, "verify": 0.0}
for e in events:
    if e["kind"] == "turn.start":
        open_turns[e["turn"]] = e
        peak = max(peak, len(open_turns))
    elif e["kind"] == "turn.end":
        s = open_turns.pop(e["turn"], None)
        d = e.get("data") or {}
        kind = d.get("kind") or ((s.get("data") or {}).get("kind", "") if s else "")
        if not kind:
            sys.exit(f"turn.end of {e['turn']} carries no turn kind")
        dur = (at(e) - at(s)).total_seconds() if s else 0
        busy[kind] = busy.get(kind, 0) + dur
        turns[e["turn"]] = {"manager": e.get("manager"), "pass": e.get("pass"), "kind": kind, "start": rel(s) if s else None, "seconds": round(dur, 1),
                            "status": "stopped" if d.get("stopped") else "ended", "cost": round((d.get("usage") or {}).get("costUSD", 0), 3), "reason": (d.get("reason") or "")[:100]}

run_report = report.get("run") or {}
if run_report and "passes" not in run_report:
    sys.exit(f"{report_path}: the run report has no 'passes' field; it is not a recursive-passes report")
if not run_report:
    print(f"{report_path}: no run in the report; report-derived fields stay empty", file=sys.stderr)
usage = run_report.get("usage") or {}
summary = {
    "run": run.name,
    "status": report.get("status"), "runStatus": run_report.get("status"), "reason": run_report.get("reason"),
    "publication": (report.get("publication") or {}).get("Outcome"),
    "evidenceSeconds": rel(events[-1]),
    "passes": [{"id": p["id"], "purpose": p["purpose"], "review": p["review"], "verdict": p.get("verdict"), "sealed": p["sealed"],
                "checks": [(c["name"], c["passed"]) for c in p.get("checks") or []],
                "turns": sum(1 for t in run_report.get("turns") or [] if t["pass"] == p["id"]),
                "costUSD": round(sum(t["usage"].get("costUSD", 0) for t in run_report.get("turns") or [] if t["pass"] == p["id"]), 3)}
               for p in run_report.get("passes") or []],
    "turns": len(turns), "executeTurns": sum(1 for t in turns.values() if t["kind"] == "execute"),
    "verifyTurns": sum(1 for t in turns.values() if t["kind"] == "verify"),
    "peakConcurrentTurns": peak, "agentSeconds": {k: round(v) for k, v in busy.items()},
    "waitReports": sum(1 for e in events if e["kind"] == "report.wait"),
    "messages": sum(1 for e in events if e["kind"] == "message"),
    "questions": sum(1 for e in events if e["kind"] == "question.asked"),
    "findings": {"total": len(run_report.get("findings") or []),
                 "blocking": sum(1 for f in run_report.get("findings") or [] if f["blocking"]),
                 "openBlocking": sum(1 for f in run_report.get("findings") or [] if f["blocking"] and not f["closed"])},
    "usage": usage,
}
print(json.dumps(summary, indent=1, default=str))
print("\nTimeline")
for e in events:
    k = e["kind"]
    d = e.get("data") or {}
    if k in ("pass.start", "pass.sealed", "candidate", "gate", "outcome", "question.asked", "question.answered", "manager.disabled", "problem.raised"):
        print(f"{rel(e):7}s {k:14} {e.get('pass', '')} {json.dumps(d)[:150]}")
    elif k == "check":
        print(f"{rel(e):7}s check          {e.get('pass', '')} {d.get('name')} passed={d.get('passed')}")
    elif k == "message":
        print(f"{rel(e):7}s message        {d['from']} -> {d['to']}: {d['text'][:170]!r}")
    elif k == "report.progress":
        print(f"{rel(e):7}s progress       {e['manager']}: {d['summary'][:170]!r}")
    elif k == "report.wait":
        print(f"{rel(e):7}s wait           {e['manager']} for {d['for']}: {d['reason'][:140]!r}")
print("\nTurns")
for tid, t in sorted(turns.items(), key=lambda x: x[1]["start"] or 0):
    print(f"  {t['start']:7}s +{t['seconds']:6}s {t['kind']:7} {t['manager']:18} {t['pass'] or '':12} {t['status'] or '':9} ${t['cost']} {t['reason']}")
print("\nFindings")
for f in run_report.get("findings") or []:
    print(f"  {f['id']:4} {f['pass']} owner={f['owner']:18} by={f['raisedBy']:18} blocking={f['blocking']!s:5} closed={f['closed']!s:5} {f['obligation'][:110]}")

"""Summarize Extract attempts as one JSON line each. Usage: rx_collect.py ATTEMPT_DIR...

Reads only the attempt's own files: timing.txt, binding.txt, the Extract report,
the owner log, the published model, and, when present, the rebuild's Apply
report and the oracle summary."""
import json
import pathlib
import statistics
import sys


def seconds(turn):
    from datetime import datetime
    parse = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00")[:26] + "+00:00")
    return (parse(turn["ended"]) - parse(turn["started"])).total_seconds()


def phase(p):
    turns = p.get("turns", [])
    verify = [seconds(t) for t in turns if t["kind"] == "verify"]
    rounds = p.get("rounds", [])
    return {
        "rounds": len(rounds),
        "green": [r.get("green") for r in rounds],
        "execute_turns": sum(t["kind"] == "execute" for t in turns),
        "verify_turns": len(verify),
        "verify_seconds_median": round(statistics.median(verify)) if verify else None,
        "verify_provider_turns_median": statistics.median([t.get("usage", {}).get("providerTurns", 0) for t in turns if t["kind"] == "verify"]) if verify else None,
        "findings_blocking": sum(1 for f in p.get("findings", []) if f.get("blocking")),
        "findings_advisory": sum(1 for f in p.get("findings", []) if not f.get("blocking")),
        "questions": len(p.get("questions", [])),
    }


def kv(path):
    out = {}
    if path.is_file():
        for line in path.read_text().splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                out[k.strip()] = v.split(" ")[0]
    return out


for arg in sys.argv[1:]:
    a = pathlib.Path(arg)
    timing, binding = kv(a / "timing.txt"), kv(a / "binding.txt")
    row = {"attempt": a.name, "variant": binding.get("variant"), "extract_exit": timing.get("extract_exit"),
           "extract_seconds": int(timing["extract_end"]) - int(timing["extract_start"]) if "extract_end" in timing else None}
    rep = a / "out" / "extract-x1.json"
    if rep.is_file() and rep.stat().st_size:
        x = json.loads(rep.read_text())
        row.update({"status": x.get("status"), "modelComplete": x.get("modelComplete"),
                    "structure": phase(x.get("structure", {})), "model": phase(x.get("model", {})),
                    "deviations": [{"owner": f.get("owner"), "obligation": f.get("obligation")[:160]} for f in x.get("deviations", [])],
                    "openQuestions": len(x.get("openQuestions", []))})
    owner = a / "out" / "owner.log"
    row["owner_answers"] = owner.read_text().splitlines() if owner.is_file() else []
    model = a / "repo" / ".markitect" / "model"
    if model.is_dir():
        files = [p for p in model.rglob("*.yaml")]
        row["packages"] = sorted({str(p.parent.relative_to(model)) for p in files})
        row["definitions"] = len(files)
        row["model_lines"] = sum(len(p.read_text().splitlines()) for p in files)
    b1 = a / "out" / "apply-b1.json"
    if b1.is_file() and b1.stat().st_size:
        b = json.loads(b1.read_text())
        run = b.get("run", {})
        row["rebuild"] = {"status": b.get("status"), "publication": (b.get("publication") or {}).get("Outcome"),
                          "rounds": len(run.get("rounds", [])) if isinstance(run, dict) else None,
                          "seconds": int(timing["rebuild_end"]) - int(timing["rebuild_start"]) if "rebuild_end" in timing else None}
    oracle = a / "assess-rebuild" / "summary.json"
    if oracle.is_file() and oracle.stat().st_size:
        o = json.loads(oracle.read_text())
        row["oracle"] = {k: o.get(k) for k in ("built", "passed", "cases_passed", "cases", "discrimination", "errors")}
    elif "oracle" in timing:
        row["oracle"] = timing["oracle"]
    print(json.dumps(row))

"""Summarize Extract attempts as one JSON line each. Usage: rx_collect.py ATTEMPT_DIR...

Reads only the attempt's own files: timing.txt, binding.txt, the Extract report,
the owner log, the published model, and, when present, the rebuild's Apply
report and the oracle summary."""
import json
import pathlib
import re
import statistics
import sys
from datetime import datetime


def parse_time(text):
    """Parse Go's RFC3339Nano: any number of fraction digits, Z or an offset."""
    m = re.fullmatch(r"(.+?)(?:\.(\d+))?(Z|[+-]\d\d:\d\d)", text)
    base, fraction, offset = m.groups()
    return datetime.fromisoformat(base + "." + (fraction or "").ljust(6, "0")[:6] + offset.replace("Z", "+00:00"))


def seconds(turn):
    return (parse_time(turn["ended"]) - parse_time(turn["started"])).total_seconds()


def deviation(f):
    """One report deviation, whichever shape its report has: the Definition with
    owners, raisers and observed texts, or an older owner and obligation."""
    if "definition" in f:
        return {"definition": f.get("definition"), "owners": f.get("owners", []), "raisedBy": f.get("raisedBy", []),
                "observed": [str(o)[:160] for o in f.get("observed", [])]}
    return {"owner": f.get("owner"), "obligation": (f.get("obligation") or "")[:160]}


def phase(p):
    """One Extract phase's run report, summarized per Pass. An absent phase is None."""
    if not p:
        return None
    if "passes" not in p:
        sys.exit("a phase report has no 'passes' field; it is not a recursive-passes report")
    turns = p.get("turns", [])
    verify = [seconds(t) for t in turns if t["kind"] == "verify"]
    passes = [{"id": q["id"], "purpose": q["purpose"], "review": q["review"], "verdict": q.get("verdict"), "sealed": q["sealed"],
               "checks": [(c["name"], c["passed"]) for c in q.get("checks") or []],
               "turns": sum(1 for t in turns if t["pass"] == q["id"])} for q in p["passes"]]
    return {
        "passes": passes,
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
                    "structure": phase(x.get("structure")), "model": phase(x.get("model")),
                    "deviations": [deviation(f) for f in x.get("deviations", [])],
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
        run = b.get("run")
        if run is not None and "passes" not in run:
            sys.exit(f"{b1}: the run report has no 'passes' field; it is not a recursive-passes report")
        row["rebuild"] = {"status": b.get("status"), "publication": (b.get("publication") or {}).get("Outcome"),
                          "passes": [(q["purpose"], q.get("verdict"), q["sealed"]) for q in run["passes"]] if run else None,
                          "seconds": int(timing["rebuild_end"]) - int(timing["rebuild_start"]) if "rebuild_end" in timing else None}
    oracle = a / "assess-rebuild" / "summary.json"
    if oracle.is_file() and oracle.stat().st_size:
        o = json.loads(oracle.read_text())
        row["oracle"] = {k: o.get(k) for k in ("built", "passed", "cases_passed", "cases", "discrimination", "errors")}
    elif "oracle" in timing:
        row["oracle"] = timing["oracle"]
    print(json.dumps(row))

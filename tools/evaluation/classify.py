"""The failure classifier: one pre-coded rule, the same for every variant.

  python -I -B classify.py RUN_DIR [--signatures FILE --variant NAME]

It sorts one run, from its suite.json and what the variant left in RUN_DIR, into:

- void: a technical failure. The run is recorded and relaunched, and it never
  enters the results. Three kinds:
  - infrastructure: the sandbox did not stand up, its container never started,
    its gateway stopped, it left residue, the gateway refused a server name in
    the provider's own domains, which its CLI needs (this also stops the
    series), the suite wrote no record, or the variant command could not be
    started at all;
  - provider: a provider session's last response is an API error for capacity,
    a rate or usage limit, a server error or a refused login, and the run did
    not end normally (no result, a non-zero exit or a timeout);
  - variant runtime: a signature of the variant's own runtime machinery,
    written into the series protocol before the series, matches the run.
- invalid: the run did not take place under the evaluation's conditions (host
  mode; in native mode, a container of the variant outside the evaluation
  network, or hidden material on the machine) or reports another model than
  the evaluation's. It is not counted.
- result: everything else, including a wrong or incomplete result, a timeout, a
  missing result, a crash of a CLI and a deliberate stop such as a model
  blocker.

A signature file holds, per variant, a list of signatures written before the
series: {"variants": {"M": [{"id": "...", "path": "reported.cause", "pattern":
"regex"}]}}. `path` is a dotted path into suite.json; the signature matches
when the value there, as text, matches the pattern. Its author states in the
protocol that no agent output (a handoff, a tool call, a report or a blocker)
leads to it.
"""
import argparse
import json
import pathlib
import re
import sys

# The `error` field of a Claude Code API error message, and the HTTP status of a stream-json result, that a provider
# causes: capacity, rate or usage limits, server errors and a refused login.
# `billing_error` and `unknown` (as for a dropped connection) are in no recorded run yet.
PROVIDER_ERRORS = {"rate_limit", "server_error", "overloaded", "overloaded_error", "api_error", "authentication_failed", "billing_error",
                   "unknown"}
PROVIDER_STATUS = {401, 403, 429, 500, 502, 503, 504, 529}
# The provider's own domains: a refused name under one of them is an endpoint its CLI needs and the series lacks.
# Any other refused name, such as an agent trying GitHub, is the variant's work.
PROVIDER_DOMAINS = {"claude": ("anthropic.com", "claude.ai", "claude.com"), "codex": ("openai.com", "chatgpt.com")}
# The variant command never ran: the interpreter or the engine could not start its program.
# The conditions a counted run takes place under: the sandbox, or a trusted variant host natively with its containers
# on the same egress-limited network (native.py).
COUNTED = ("sandbox", "native")
NOT_STARTED = re.compile(r"can't open file '[^']*'|executable file not found|exec format error|OCI runtime (create|exec) failed")


def load(path):
    try:
        value = json.loads(pathlib.Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return None
    return value if isinstance(value, dict) else None


def at(record, dotted):
    value = record
    for part in dotted.split("."):
        if not isinstance(value, dict) or part not in value:
            return None
        value = value[part]
    return value


def session_errors(root):
    """The provider error that ended each top-level Claude Code session below root, if any.

    A session ends in an error when its last assistant message is an API error message, or when its stream-json
    result reports an API error status. Subagent transcripts do not count: the session that ran them decides.
    """
    found = []
    for path in sorted(pathlib.Path(root).rglob("*.jsonl")):
        if path.is_symlink() or not path.is_file() or "subagents" in path.parts:
            continue
        last, result = None, None
        for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                entry = json.loads(raw)
            except ValueError:
                continue
            if not isinstance(entry, dict) or entry.get("isSidechain") or entry.get("parent_tool_use_id"):
                continue
            if entry.get("type") == "assistant":
                last = entry
            elif entry.get("type") == "result":
                result = entry
        if last and last.get("isApiErrorMessage") and last.get("error") in PROVIDER_ERRORS:
            found.append({"file": path.relative_to(root).as_posix(), "error": last["error"], "apiError": last.get("apiError")})
        elif result and result.get("is_error") and result.get("api_error_status") in PROVIDER_STATUS:
            found.append({"file": path.relative_to(root).as_posix(), "error": f"status {result['api_error_status']}"})
    return found


def classify(run_dir, signatures=None, variant=None):
    """One run's class, kind, the evidence for it and whether the series must stop."""
    run_dir = pathlib.Path(run_dir)
    record = load(run_dir / "suite.json")

    def outcome(cls, kind=None, evidence=None, stop=False):
        return {"class": cls, "kind": kind, "evidence": evidence, "stopsSeries": stop}

    if record is None:
        return outcome("void", "infrastructure", "the suite wrote no readable suite.json")
    box = record.get("sandbox") or {}
    domains = PROVIDER_DOMAINS.get(str(record.get("model", "")).partition(":")[0], ())
    needed = sorted(name for name in (box.get("egress") or {}).get("refused") or {}
                    if any(name == d or name.endswith("." + d) for d in domains))
    if needed:
        return outcome("void", "infrastructure", "the gateway refused the provider's " + ", ".join(needed), True)
    if box.get("escapes"):
        # A container of the variant left the evaluation's network: the run did not take place under its conditions.
        return outcome("invalid", None, "a container outside the evaluation network: " + ", ".join(
            str(c.get("name") or c.get("id")) for c in box["escapes"]))
    if box.get("hiddenPresent"):
        return outcome("invalid", None, "hidden material was on the machine: " + ", ".join(box["hiddenPresent"]))
    if record.get("conditions") in COUNTED and not record.get("validForCounting"):
        cause = box.get("error") or ("residue: " + ", ".join(box.get("residue") or []) if box.get("residue") else "not valid for counting")
        return outcome("void", "infrastructure", cause)
    variant_out = run_dir / "variant"
    stderr = run_dir / "variant.stderr.log"
    wrote_nothing = not variant_out.is_dir() or not any(variant_out.iterdir())
    if wrote_nothing and stderr.is_file():
        match = NOT_STARTED.search(stderr.read_text(encoding="utf-8", errors="replace"))
        if match:
            return outcome("void", "infrastructure", "the variant command never ran: " + match.group(0))
    ended_normally = record.get("delivered") and record.get("exitCode") == 0 and not record.get("timedOut")
    if not ended_normally and variant_out.is_dir():
        errors = session_errors(variant_out)
        if errors:
            return outcome("void", "provider", errors)
    for signature in ((signatures or {}).get("variants") or {}).get(variant) or []:
        value = at(record, signature["path"])
        if value is not None and re.search(signature["pattern"], value if isinstance(value, str) else json.dumps(value)):
            return outcome("void", "variant runtime", {"signature": signature["id"], "path": signature["path"]})
    if record.get("conditions") not in COUNTED:
        return outcome("invalid", None, "host mode")
    if record.get("modelMatches") is False:
        return outcome("invalid", None, "the variant reports another model")
    return outcome("result")


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("run")
    p.add_argument("--signatures", help="the series' variant runtime signatures")
    p.add_argument("--variant", help="the variant's name in the signature file")
    a = p.parse_args(argv)
    signatures = load(a.signatures) if a.signatures else None
    if a.signatures and signatures is None:
        p.error(f"{a.signatures} is not a JSON object")
    print(json.dumps(classify(a.run, signatures, a.variant), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())

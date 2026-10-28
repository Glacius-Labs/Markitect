"""One fresh reviewer session of the blinded code review: a headless Claude Code session that may only read.

  python -I -B reviewer.py [--claude BIN] [--credentials FILE] MATERIAL_DIR OUT_DIR

`suite.py review` runs it in the evaluation sandbox, the way it runs a
variant: MATERIAL_DIR (read-only at /task) holds review.json, the blinded token
directory, requirements.md and rubric.md. review.json carries the prompt, in
which TOKEN_DIR, REQUIREMENTS and RUBRIC stand for those paths, and the
session's time. The session works on a private copy, with the Read, Glob and
Grep tools only, the evaluation's model from EVALUATION_MODEL and the
baseline's isolated configuration (claude_code.py): no user settings, plugins,
CLAUDE.md, memory or MCP servers, and an access-only login copy.

It writes OUT_DIR/answer.json (the JSON object of the session's final answer),
run.json, the session's stream and its transcripts.
"""
import argparse
import json
import os
import pathlib
import shutil
import sys
import tempfile

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import claude_code  # noqa: E402

TOOLS = ["Read", "Glob", "Grep"]
DENIED = ["Bash", "Edit", "Write", "NotebookEdit", "WebFetch", "WebSearch", "Agent", "Task"]


def answer_object(text):
    """The JSON object a reviewer answered with: the whole text, a fenced block or the outermost braces."""
    text = (text or "").strip()
    candidates = [text]
    if "```" in text:
        candidates += [part.split("\n", 1)[1] if "\n" in part else part for part in text.split("```")[1::2]]
    if "{" in text:
        candidates.append(text[text.index("{"):text.rindex("}") + 1])
    for candidate in candidates:
        try:
            value = json.loads(candidate)
        except ValueError:
            continue
        if isinstance(value, dict):
            return value
    return None


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--claude", default=shutil.which("claude") or "claude")
    parser.add_argument("--credentials", default=os.environ.get("EVALUATION_LOGIN") or str(pathlib.Path.home() / ".claude" / ".credentials.json"))
    parser.add_argument("material")
    parser.add_argument("out")
    a = parser.parse_args(argv)
    material, out = pathlib.Path(a.material), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    record = {"provider": "claude", "model": None, "status": "error", "cause": None}
    try:
        provider, _, model = os.environ.get("EVALUATION_MODEL", "").partition(":")
        if provider != "claude" or not model:
            raise claude_code.VariantError(f"EVALUATION_MODEL must be claude:MODEL_ID, not {os.environ.get('EVALUATION_MODEL')!r}")
        record["requestedModel"] = model
        spec = json.loads((material / "review.json").read_text(encoding="utf-8"))
        seconds = int(spec["deadlineSeconds"]) - claude_code.MARGIN
        if seconds <= 0:
            raise claude_code.VariantError(f"a deadline of {spec['deadlineSeconds']} s leaves no time for a session")
        login, secrets = claude_code.session_login(pathlib.Path(a.credentials).read_bytes(), seconds)
        with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp)
            work, home, config, hooks = tmp / "review", tmp / "home", tmp / "config", tmp / "no-hooks"
            shutil.copytree(material, work, ignore=lambda d, names: ["review.json"] if pathlib.Path(d) == material else [])
            for d in (home, config, hooks):
                d.mkdir()
            prompt = spec["prompt"]
            for name, path in (("TOKEN_DIR", work / spec["token"]), ("REQUIREMENTS", work / "requirements.md"), ("RUBRIC", work / "rubric.md")):
                prompt = prompt.replace(name, str(path))
            (tmp / "prompt.txt").write_text(prompt, encoding="utf-8")
            env = claude_code.base_env(home, hooks)
            try:
                (config / ".credentials.json").write_bytes(login)
                argv = [a.claude, "-p", "--model", model, "--output-format", "stream-json", "--verbose",
                        "--strict-mcp-config", "--setting-sources", "project", "--allowedTools", *TOOLS, "--disallowedTools", *DENIED]
                record.update(claude_code.run_session(argv, tmp / "prompt.txt", work, dict(env, CLAUDE_CONFIG_DIR=str(config)),
                                                      out / "session.stream.jsonl", out / "session.stderr.log", seconds))
            finally:
                (config / ".credentials.json").unlink(missing_ok=True)
            facts = claude_code.stream_facts(out / "session.stream.jsonl")
            record.update(model=facts["model"], sessionId=facts["sessionId"], usage=claude_code.usage(facts["result"]),
                          sessionResult=(facts["result"] or {}).get("subtype"))
            if (config / "projects").is_dir():
                shutil.copytree(config / "projects", out / "transcripts")
        answer = answer_object((facts["result"] or {}).get("result"))
        if answer is not None:
            (out / "answer.json").write_text(json.dumps(answer, indent=2) + "\n", encoding="utf-8")
        record["status"] = "timed_out" if record["timedOut"] else ("completed" if record["exitCode"] == 0 and answer is not None else "failed")
        if claude_code.contains_secret(out, secrets):
            for p in sorted(out.rglob("*"), reverse=True):
                p.rmdir() if p.is_dir() else p.unlink()
            record["quarantine"], record["status"] = True, "quarantined"
    except (claude_code.VariantError, OSError, ValueError, KeyError) as e:
        record["cause"] = f"{type(e).__name__}: {e}"
    (out / "run.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if record["status"] == "completed" else 1


if __name__ == "__main__":
    sys.exit(main())

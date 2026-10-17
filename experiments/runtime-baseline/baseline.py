"""Single-agent baseline: one Claude Code or Codex session realizes library M1
in the hardened runtime container, then the independent oracle assesses it.

This exercises the runtime image, a long real session and the assessment
harness before the Manager process exists. It is not an Apply and not the
Manager organization. The Agent receives only the canonical model and the
software vocabulary, never the evaluation contract.

Usage:
  python -I baseline.py [--provider claude|codex] [--budget 5] [--out DIR]
Claude Code runs Haiku and Codex runs Luna, both with high effort (owner's
choice). Claude reads CLAUDE_CODE_OAUTH_TOKEN from
~/.markitect/runtime.env. Codex receives a copy of the dedicated
runtime login (volume mk-codex-runtime-auth) without its refresh token.
Credentials are never printed.
"""
import argparse
import datetime
import io
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tarfile

HERE = pathlib.Path(__file__).resolve().parent
# Runs hold transcripts and Agent-produced code; keep them outside the repository.
RUNS = pathlib.Path(os.environ.get("MARKITECT_BASELINE_RUNS") or pathlib.Path.home() / ".markitect" / "durchstich" / "runs")
REPO = str(pathlib.Path(__file__).resolve().parents[2])
ENGINE = os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker"
CREDENTIALS = pathlib.Path(os.environ.get("MARKITECT_RUNTIME_ENV") or pathlib.Path.home() / ".markitect" / "runtime.env")
CODEX_AUTH_VOLUME = "mk-codex-runtime-auth"
CLAUDE_MODEL, CODEX_MODEL = "claude-haiku-5-5", "gpt-6-luna"
EXAMPLE = "docs/design/canonical-model/library-example"

PROMPT = """You are implementing a small library management program from its canonical model.

The model is in /work/model: YAML Definitions with Markdown bodies and [[links]]. /work/model/model/library/callable-interface.yaml defines the program boundary, the build layout, the wire format and every operation. The Kinds come from the vocabulary in /work/vocabulary.

Create the complete Go realization in /work/realization exactly as the callable interface requires: module realization/go.mod, entry point realization/cmd/library, Go standard library only, with tests. From /work/realization, `go build ./cmd/library` and `go test ./...` must pass.

Follow every rule in the model. Do not invent operations, fields or behavior the model does not state. Work autonomously until the realization is complete and its tests pass, then reply with a short summary."""


def load_credential():
    for line in CREDENTIALS.read_text(encoding="utf-8").splitlines():
        if line.startswith("CLAUDE_CODE_OAUTH_TOKEN="):
            return line.split("=", 1)[1].strip()
    sys.exit("no CLAUDE_CODE_OAUTH_TOKEN in runtime.env")


def model_tar(rev):
    """The M1 model and vocabulary as a tar stream rooted at /work."""
    archive = subprocess.run(["git", "-C", REPO, "archive", rev, f"{EXAMPLE}/m1", f"{EXAMPLE}/vocabulary"],
                             check=True, capture_output=True).stdout
    out = io.BytesIO()
    with tarfile.open(fileobj=io.BytesIO(archive)) as src, tarfile.open(fileobj=out, mode="w") as dst:
        for member in src.getmembers():
            rel = member.name[len(EXAMPLE) + 1:]
            if not rel:
                continue
            rel = "model" + rel[2:] if rel.startswith("m1") else rel
            data = src.extractfile(member) if member.isfile() else None
            member.name = rel
            member.uid = member.gid = 1000
            member.mode = 0o755 if member.isdir() else 0o644
            dst.addfile(member, data)
    return out.getvalue()


def docker(*args, **kw):
    return subprocess.run([ENGINE, *args], **kw)


def issue_codex_login(image, home):
    """Write the runtime login without its refresh token into a provider home."""
    raw = docker("run", "--rm", "--user", "1000:1000", "--mount", f"type=volume,src={CODEX_AUTH_VOLUME},dst=/auth,readonly",
                 "--entrypoint", "/bin/cat", image, "/auth/auth.json", check=True, capture_output=True).stdout
    login = json.loads(raw)
    login["tokens"]["refresh_token"] = ""
    docker("run", "-i", "--rm", "--user", "1000:1000", "--mount", f"type=volume,src={home},dst=/home/agent",
           "--entrypoint", "/bin/sh", image, "-c", "mkdir -p /home/agent/.codex && umask 077 && cat > /home/agent/.codex/auth.json",
           input=json.dumps(login).encode(), check=True, capture_output=True)


def provider_command(provider, budget):
    if provider == "claude":
        return ["claude", ["-p", PROMPT, "--output-format", "stream-json", "--verbose",
                           "--tools", "Bash,Read,Write,Edit,Glob,Grep",
                           "--allowedTools", "Bash", "Read", "Write", "Edit", "Glob", "Grep",
                           "--permission-mode", "dontAsk", "--max-budget-usd", str(budget), "--no-session-persistence",
                           "--model", CLAUDE_MODEL, "--effort", "high"]]
    return ["codex", ["exec", "--json", "--skip-git-repo-check", "--ephemeral", "--sandbox", "danger-full-access",
                      "--model", CODEX_MODEL, "-c", "model_reasoning_effort=\"high\"",
                      "--disable", "apps", "--disable", "memories", "--disable", "multi_agent",
                      "--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
                      "-C", "/work", PROMPT]]


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--provider", choices=["claude", "codex"], default="claude")
    p.add_argument("--budget", type=float, default=5.0)
    p.add_argument("--image", default="markitect-runtime-go:dev")
    p.add_argument("--rev", default="origin/main")
    p.add_argument("--out")
    p.add_argument("--timeout", type=int, default=45 * 60)
    a = p.parse_args()

    stamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    out = pathlib.Path(a.out or RUNS / f"baseline-{a.provider}-{stamp}").resolve()
    out.mkdir(parents=True, exist_ok=True)
    commit = subprocess.run(["git", "-C", REPO, "rev-parse", a.rev], check=True, capture_output=True, text=True).stdout.strip()
    label = f"mk-durchstich-{a.provider}-{stamp}"
    home, work = label + "-home", label + "-work"
    for v in (home, work):
        docker("volume", "create", "--label", f"io.markitect.store={label}", v, check=True, capture_output=True)
    env = dict(os.environ, CLAUDE_CODE_OAUTH_TOKEN=load_credential()) if a.provider == "claude" else dict(os.environ)
    model = CLAUDE_MODEL if a.provider == "claude" else CODEX_MODEL
    summary = {"label": label, "provider": a.provider, "model": model, "effort": "high",
               "model_commit": commit, "budget_usd": a.budget, "image": a.image}
    try:
        docker("run", "-i", "--rm", "--user", "1000:1000", "--entrypoint", "/bin/tar",
               "--mount", f"type=volume,src={work},dst=/work", a.image, "-x", "-C", "/work",
               input=model_tar(commit), check=True, capture_output=True)
        if a.provider == "codex":
            issue_codex_login(a.image, home)
        entrypoint, args = provider_command(a.provider, a.budget)
        credential = ["-e", "CLAUDE_CODE_OAUTH_TOKEN"] if a.provider == "claude" else []
        started = datetime.datetime.now()
        with open(out / "transcript.jsonl", "wb") as transcript, open(out / "stderr.log", "wb") as err:
            run = docker(
                "run", "-i", "--rm", "--name", label, "--init", "--read-only", "--cap-drop", "ALL",
                "--security-opt", "no-new-privileges", "--user", "1000:1000",
                "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=536870912",
                "--cpus", "2", "--memory", "4294967296", "--pids-limit", "512",
                "--label", f"io.markitect.store={label}",
                "--mount", f"type=volume,src={home},dst=/home/agent",
                "--mount", f"type=volume,src={work},dst=/work",
                "-w", "/work", *credential, "--entrypoint", entrypoint, a.image, *args,
                env=env, stdin=subprocess.DEVNULL, stdout=transcript, stderr=err, timeout=a.timeout)
        summary["exit_code"] = run.returncode
        summary["minutes"] = round((datetime.datetime.now() - started).total_seconds() / 60, 1)
        for line in (out / "transcript.jsonl").read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if event.get("type") == "result":
                summary.update({k: event.get(k) for k in ("subtype", "is_error", "num_turns", "total_cost_usd")})
                summary["result"] = (event.get("result") or "")[:2000]
            if event.get("type") == "turn.completed":
                summary["usage"] = event.get("usage")
            if event.get("type") == "item.completed" and (event.get("item") or {}).get("type") == "agent_message":
                summary["result"] = (event["item"].get("text") or "")[:2000]
        tar = docker("run", "--rm", "--user", "1000:1000", "--entrypoint", "/bin/tar",
                     "--mount", f"type=volume,src={work},dst=/work", a.image, "-c", "-C", "/work", "realization",
                     capture_output=True)
        if tar.returncode == 0:
            with tarfile.open(fileobj=io.BytesIO(tar.stdout)) as t:
                t.extractall(out, filter="data")
            assessment = subprocess.run([sys.executable, "-I", str(HERE / "assess.py"), "--realization",
                                         str(out / "realization"), "--profile", "m1", "--out", str(out / "assessment"),
                                         "--label", f"baseline-{a.provider}-A1", "--rev", commit], capture_output=True, text=True)
            summary["assessment"] = json.loads(assessment.stdout) if assessment.stdout.strip().startswith("{") else assessment.stdout + assessment.stderr
        else:
            summary["assessment"] = "no realization directory produced"
    except subprocess.TimeoutExpired:
        summary["exit_code"] = "timeout"
        docker("rm", "--force", label, capture_output=True)
    finally:
        for v in (home, work):
            docker("volume", "rm", "-f", v, capture_output=True)
        (out / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
        print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()

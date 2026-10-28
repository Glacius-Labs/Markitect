"""A scripted stand-in for a native arm: a host process whose "agent" runs in a container of its own on the suite's network.

  python -I -B native_standin.py --image IMAGE_ID [--probe] TASK_DIR OUT_DIR [STATE_DIR]

Run it through `suite.py run --native IMAGE_ID ...`, which hands it the
network, the provider endpoints, the label and the engine (native.py). It
stands in for the concept 2.0 arm until that arm exists, so that the native
mode is tested the same way on Windows and on Linux. No model is used.

It clones the task's start on the host and starts one Run container from the
sandbox image IMAGE_ID, as the 2.0 arm starts its agents: on EVALUATION_NETWORK
with the provider endpoints at the gateway and a resolver that resolves
nothing, hardened like the sandbox, with the task at /task, the work at /out,
this folder at /variant and the access-only login at /login/credentials.json.
The container's "agent" acts as the first line of task.md says:

  change    writes STANDIN.md into the work; the arm commits and delivers it
  nothing   changes nothing; the arm delivers its start, as a Run that does not land
  escape    also starts a container on the engine's default network, which the suite must flag
  leave     leaves its Run container behind, which the suite must count as residue

With --probe the container runs the escape probe instead (escape_probe.py):
the probe as an agent in a Run container. The arm then writes the probe's
report as its own run.json, delivers no result and exits 1 if anything escaped.
"""
import argparse
import json
import os
import pathlib
import shutil
import subprocess
import sys
import uuid

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import sandbox as sandboxing  # noqa: E402

IDENTITY = ("Native Stand-In", "standin@markitect.invalid")
DATE = "2026-10-28T00:00:00Z"
# The scripted agent: python3 -c AGENT TASK_DIR OUT_DIR.
AGENT = ("import pathlib, sys; mode = pathlib.Path(sys.argv[1], 'task.md').read_text().splitlines()[0]; "
         "mode == 'nothing' or pathlib.Path(sys.argv[2], 'repo', 'STANDIN.md').write_text('The stand-in agent was here: ' + mode + chr(10))")


def git(repo, *args, env=None):
    return subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], check=True, capture_output=True, text=True,
                          env=env).stdout.strip()


def run_container(engine, image, name, work, task, login, probe):
    hosts = [f"--add-host={pair}" for pair in os.environ["EVALUATION_HOSTS"].split(",") if pair]
    mounts = [(task, "/task", True), (work, "/out", False), (HERE, "/variant", True), (login, "/login/credentials.json", True)]
    argv = [*engine, "run", "--name", name, "--label", os.environ["EVALUATION_LABEL"], "--init", "--network", os.environ["EVALUATION_NETWORK"],
            "--dns", os.environ["EVALUATION_DNS"], *hosts, *sandboxing.hardening(sandboxing.container_user()),
            "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=268435456", "--memory", "1073741824", "--pids-limit", "256",
            "-e", "HOME=/tmp", "-e", f"EVALUATION_MODEL={os.environ['EVALUATION_MODEL']}", "-e", "EVALUATION_LOGIN=/login/credentials.json"]
    for src, dst, ro in mounts:
        argv += ["--mount", f"type=bind,src={sandboxing.host_path(src)},dst={dst}" + (",readonly" if ro else "")]
    endpoint = os.environ["EVALUATION_HOSTS"].split(",")[0].rpartition(":")[0]
    command = (["python3", "-I", "-B", "/variant/escape_probe.py", "--endpoint", endpoint, "/task", "/out"] if probe
               else ["python3", "-I", "-B", "-c", AGENT, "/task", "/out"])
    return subprocess.run([*argv, "--workdir", "/tmp", image, *command], capture_output=True, text=True).returncode


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--image", required=True, help="the sandbox image ID the Run container starts from")
    p.add_argument("--probe", action="store_true", help="run the escape probe as the agent")
    p.add_argument("task")
    p.add_argument("out")
    p.add_argument("state", nargs="?")
    a = p.parse_args(argv)
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    engine = json.loads(os.environ["EVALUATION_ENGINE"])
    model = os.environ["EVALUATION_MODEL"]
    mode = (task / "task.md").read_text(encoding="utf-8").splitlines()[0].strip()
    work = out / "work"
    work.mkdir()
    name = "standin-" + uuid.uuid4().hex[:8]
    report = {"provider": model.partition(":")[0], "model": model.partition(":")[2], "standIn": True, "mode": "probe" if a.probe else mode}
    commit_env = dict(os.environ, GIT_AUTHOR_NAME=IDENTITY[0], GIT_AUTHOR_EMAIL=IDENTITY[1], GIT_COMMITTER_NAME=IDENTITY[0],
                    GIT_COMMITTER_EMAIL=IDENTITY[1], GIT_AUTHOR_DATE=DATE, GIT_COMMITTER_DATE=DATE, GIT_CONFIG_NOSYSTEM="1")
    if not a.probe:
        start = json.loads((task / "task.json").read_text(encoding="utf-8"))["start"]["commit"]
        subprocess.run(["git", "clone", "-q", str((task / "start.bundle").resolve()), str(work / "repo")], check=True, capture_output=True)
        git(work / "repo", "checkout", "-q", "-B", "work", start)
    try:
        report["exit"] = run_container(engine, a.image, name, work, task, os.environ["EVALUATION_LOGIN"], a.probe)
        if mode == "escape" and not a.probe:
            subprocess.run([*engine, "run", "--name", name + "-out", "--label", os.environ["EVALUATION_LABEL"], a.image, "true"],
                           capture_output=True)
            subprocess.run([*engine, "rm", "-f", name + "-out"], capture_output=True)
    finally:
        if mode != "leave":
            subprocess.run([*engine, "rm", "-f", name], capture_output=True)
    if a.probe:
        probe = json.loads((work / "run.json").read_text(encoding="utf-8")) if (work / "run.json").is_file() else {"status": "no report"}
        report.update(status=probe.get("status"), egressWorks=probe.get("egressWorks"), probe=probe.get("probe"))
        (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        shutil.rmtree(work, ignore_errors=True)
        return 0 if probe.get("status") == "contained" else 1
    repo = work / "repo"
    if mode != "nothing":
        git(repo, "add", "-A", env=commit_env)
        git(repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "Stand-in Run", env=commit_env)
    git(repo, "branch", "-f", "result", "HEAD")
    git(repo, "bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
    (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    shutil.rmtree(work, ignore_errors=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())

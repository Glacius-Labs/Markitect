"""Dedicated Codex runtime login (OQ-010 probe), separate from the desktop.

Design: the runtime login, including its refresh token, lives only in the
Docker volume `mk-codex-runtime-auth`. One maintenance container is its only
writer, so refreshes never race and the desktop login is never touched. A
session never receives the refresh token: before a run, a helper container
writes a copy of the login without its refresh token into the session's
provider home. Access tokens outlive a run, so no session needs a refresh.
(`codex login --with-access-token` expects an agent-identity token and cannot
take a ChatGPT access token.)

  python -I codex_login.py login                # device-code login: open the printed URL, enter the code
  python -I codex_login.py status               # does the runtime login work? (no secrets printed)
  python -I codex_login.py issue <home-volume>  # write a refresh-free copy into a session's provider home
"""
import json
import os
import shutil
import subprocess
import sys

ENGINE = os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker"
IMAGE = "markitect-runtime:dev"
VOLUME = "mk-codex-runtime-auth"
HARDENING = ["--rm", "--user", "1000:1000", "--cap-drop", "ALL", "--security-opt", "no-new-privileges"]


def docker(*args, **kw):
    return subprocess.run([ENGINE, *args], **kw)


def ensure_volume():
    if docker("volume", "inspect", VOLUME, capture_output=True).returncode != 0:
        docker("volume", "create", "--label", "io.markitect.credential=codex-runtime", VOLUME, check=True, capture_output=True)
        # A fresh volume at a path the image lacks is root-owned; hand it to the session user.
        for cmd in (["/bin/chown", "1000:1000", "/x"], ["/bin/chmod", "700", "/x"]):
            docker("run", "--rm", "--user", "0:0", "--mount", f"type=volume,src={VOLUME},dst=/x",
                   "--entrypoint", cmd[0], IMAGE, *cmd[1:], check=True, capture_output=True)


def maintenance(*args, interactive=False, **kw):
    flags = ["-it"] if interactive else []
    return docker("run", *flags, *HARDENING, "--mount", f"type=volume,src={VOLUME},dst=/home/agent/.codex",
                  "-e", "CODEX_HOME=/home/agent/.codex", "--entrypoint", "codex", IMAGE, *args, **kw)


def session_login():
    raw = docker("run", *HARDENING, "--mount", f"type=volume,src={VOLUME},dst=/auth,readonly",
                 "--entrypoint", "/bin/cat", IMAGE, "/auth/auth.json", capture_output=True, check=True).stdout
    login = json.loads(raw)
    login["tokens"]["refresh_token"] = ""
    return json.dumps(login)


def main():
    if len(sys.argv) < 2 or sys.argv[1] not in ("login", "status", "issue"):
        sys.exit(__doc__)
    ensure_volume()
    verb = sys.argv[1]
    if verb == "login":
        sys.exit(maintenance("login", "--device-auth", interactive=True).returncode)
    if verb == "status":
        result = maintenance("login", "status", capture_output=True, text=True)
        text = (result.stdout + result.stderr).strip()
        print("\n".join(l for l in text.splitlines() if "token" not in l.lower() and "sk-" not in l) or "no output")
        sys.exit(result.returncode)
    if len(sys.argv) != 3:
        sys.exit("issue needs the session's provider-home volume")
    result = docker("run", "-i", *HARDENING, "--mount", f"type=volume,src={sys.argv[2]},dst=/home/agent",
                    "--entrypoint", "/bin/sh", IMAGE, "-c",
                    "mkdir -p /home/agent/.codex && umask 077 && cat > /home/agent/.codex/auth.json",
                    input=session_login(), text=True, capture_output=True)
    print("issued" if result.returncode == 0 else "failed")
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()

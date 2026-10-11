# One fresh container per run. The host passes the command; no secrets in the image.
# Both outer agents ship in every image (Codex is also Markitect's inner-role runtime);
# the host tags the image with both versions.
#
# This file is the one place for the pins (README "Updating the image pins"): the base
# image by its index digest, the Debian packages by a snapshot.debian.org time not older
# than the base image, both CLIs by exact npm versions. The host (playground/image.py)
# refuses a base without @sha256:, passes SOURCE_DATE_EPOCH (a BuildKit build argument)
# from the snapshot, and compares the inventory the last step writes with
# container/inventory/codex-<v>-claude-<v>.txt.
ARG BASE_IMAGE=node:22-bookworm-slim@sha256:c3de60bf2f9dd0ac6370e6117950ff62d6e339527e7472301c9c78a017978392
FROM ${BASE_IMAGE}
ARG BASE_IMAGE
ARG DEBIAN_SNAPSHOT=20261010T203349Z

# Debian packages (debian and debian-security) from the snapshot, so a cold build installs
# the same versions. A snapshot's Release files expire: Check-Valid-Until is off. The
# snapshot service is slow at times: apt retries every download, and the step runs at
# most three times.
RUN test -n "${DEBIAN_SNAPSHOT}" \
 && rm -f /etc/apt/sources.list \
 && printf '%s\n' \
      'Types: deb' \
      "URIs: http://snapshot.debian.org/archive/debian/${DEBIAN_SNAPSHOT}" \
      'Suites: bookworm bookworm-updates' \
      'Components: main' \
      'Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg' \
      '' \
      'Types: deb' \
      "URIs: http://snapshot.debian.org/archive/debian-security/${DEBIAN_SNAPSHOT}" \
      'Suites: bookworm-security' \
      'Components: main' \
      'Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg' \
      > /etc/apt/sources.list.d/debian.sources \
 && printf '%s\n' 'Acquire::Check-Valid-Until "false";' 'Acquire::Retries "5";' \
      > /etc/apt/apt.conf.d/80markitect-playground \
 && for attempt in 1 2 3; do \
      if apt-get update \
         && apt-get install -y --no-install-recommends \
              python3 python-is-python3 git ca-certificates procps bubblewrap ripgrep util-linux tini; then \
        break; \
      fi; \
      if [ "${attempt}" = 3 ]; then echo "apt failed three times" >&2; exit 1; fi; \
      echo "apt attempt ${attempt} failed; retrying in 30 s" >&2; \
      sleep 30; \
    done \
 && rm -rf /var/lib/apt/lists/*

ARG CODEX_VERSION
RUN test -n "${CODEX_VERSION}" \
 && npm install -g "@openai/codex@${CODEX_VERSION}" \
 && npm cache clean --force \
 && codex --version \
 && codex --version | grep -qx "codex-cli ${CODEX_VERSION}"

ARG CLAUDE_VERSION=2.1.296
RUN test -n "${CLAUDE_VERSION}" \
 && npm install -g "@anthropic-ai/claude-code@${CLAUDE_VERSION}" \
 && npm cache clean --force \
 && DISABLE_AUTOUPDATER=1 claude --version \
 && DISABLE_AUTOUPDATER=1 claude --version | grep -qx "${CLAUDE_VERSION} (Claude Code)" \
 && rm -rf /root/.claude /root/.claude.json

# The base image's "node" user owns uid/gid 1000; replace it with "agent".
RUN userdel --remove node \
 && groupadd --gid 1000 agent \
 && useradd --uid 1000 --gid 1000 --create-home --home-dir /home/agent --shell /bin/bash agent

# No image-wide safe.directory: the root runner never runs Git on the agent's repo
# itself; lifecycle.py runs those Git reads as the repo owner.

ENV LANG=C.UTF-8 PYTHONUTF8=1 PYTHONDONTWRITEBYTECODE=1

# Last step: the inventory of what this image holds (format: playground/image.py). The
# host reads it after every build and compares it with the committed one.
RUN python3 -I -B - <<'PY'
import json
import os
import subprocess
from pathlib import Path


def output(*cmd):
    return subprocess.run(cmd, check=True, capture_output=True, text=True).stdout


def walk(folder, name, found):
    """name@version of the package in `folder` and of every package below its node_modules."""
    try:
        version = json.loads((folder / "package.json").read_text(encoding="utf-8")).get("version")
    except (OSError, ValueError):
        version = None
    found.add(f"{name}@{version if isinstance(version, str) else ''}")
    modules = folder / "node_modules"
    if not modules.is_dir():
        return
    for entry in sorted(modules.iterdir()):
        if entry.name.startswith(".") or not entry.is_dir():
            continue
        if entry.name.startswith("@"):
            for scoped in sorted(entry.iterdir()):
                if scoped.is_dir():
                    walk(scoped, f"{entry.name}/{scoped.name}", found)
        else:
            walk(entry, entry.name, found)


root = Path(output("npm", "root", "-g").strip())
npm = set()
for cli in ("@openai/codex", "@anthropic-ai/claude-code"):
    walk(root / cli, cli, npm)
dollar = "$"  # no dollar-brace in this script: the Dockerfile never expands it
status = " ".join(dollar + "{" + field + "}" for field in ("db:Status-Status", "Package", "Version", "Architecture"))
installed = (line.split() for line in output("dpkg-query", "-W", "-f", status + "\n").splitlines())
dpkg = sorted(" ".join(fields[1:]) for fields in installed if fields and fields[0] == "installed")
lines = ["markitect-playground-inventory 1", f"base {os.environ['BASE_IMAGE']}",
         f"snapshot {os.environ['DEBIAN_SNAPSHOT']}", f"node {output('node', '--version').strip()}"]
lines += [f"dpkg {entry}" for entry in dpkg] + [f"npm {entry}" for entry in sorted(npm)]
target = Path("/usr/share/markitect-playground/inventory.txt")
target.parent.mkdir(parents=True, exist_ok=True)
target.write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

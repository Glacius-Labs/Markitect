# One fresh container per run. The host passes the command; no secrets in the image.
# Both outer agents ship in every image (Codex is also Markitect's inner-role runtime);
# the host tags the image with both versions.
FROM node:22-bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      python3 python-is-python3 git ca-certificates procps bubblewrap ripgrep util-linux tini \
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

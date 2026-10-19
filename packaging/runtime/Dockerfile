# Markitect runtime base image: the session helper markitect-agentd and the
# pinned Codex and Claude Code CLIs. Build it from the repository root:
#
#   docker build -f packaging/runtime/Dockerfile -t markitect-runtime .
#
# A project builds its own image FROM this one and adds its toolchain. The
# Host names the resulting image by its engine image ID.

FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS agentd
WORKDIR /src
COPY go.mod go.sum ./
COPY internal/agentd ./internal/agentd
COPY cmd/markitect-agentd ./cmd/markitect-agentd
ARG MARKITECT_VERSION=development
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=${MARKITECT_VERSION}" \
      -o /out/markitect-agentd ./cmd/markitect-agentd

FROM node:22-bookworm-slim@sha256:c3de60bf2f9dd0ac6370e6117950ff62d6e339527e7472301c9c78a017978392
ARG CODEX_VERSION=0.162.0
ARG CLAUDE_CODE_VERSION=2.1.301
RUN apt-get update \
 && apt-get install --yes --no-install-recommends ca-certificates git python3 \
 && rm -rf /var/lib/apt/lists/*
RUN npm install --global --no-fund --no-audit \
      "@openai/codex@${CODEX_VERSION}" "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
 && npm cache clean --force
COPY --from=agentd /out/markitect-agentd /usr/local/bin/markitect-agentd

# Fresh named volumes mounted at these paths take over their ownership, so
# the non-root session user can write its provider home and workspace.
RUN mkdir -p /home/agent /work && chown 1000:1000 /home/agent /work

# Record what preparation binds: the CLI versions, the generated Codex App
# Server schema digest and the agentd build. The Host copies this file out of
# a created, never started container.
RUN set -eu; \
    mkdir -p /opt/markitect/codex-schema; \
    codex app-server generate-json-schema --experimental --out /opt/markitect/codex-schema >/dev/null; \
    schema=$(cd /opt/markitect/codex-schema && find . -type f -print0 | LC_ALL=C sort -z | xargs -0 sha256sum | sha256sum | cut -d' ' -f1); \
    printf 'codex=%s\nclaude=%s\ncodexSchema=sha256:%s\nagentd=%s\n' \
      "$(codex --version)" "$(claude --version)" "$schema" "$(markitect-agentd version)" \
      > /opt/markitect/runtime.env

ENV HOME=/home/agent \
    DISABLE_AUTOUPDATER=1 \
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/markitect-agentd", "serve"]

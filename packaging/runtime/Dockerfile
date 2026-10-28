# Markitect Manager image: the session helper markitect-agentd, the markitect
# command for reading the model, Git, and the pinned Claude Code and Codex
# CLIs. Every maker and checker of a Run runs in a container of this image or
# of a project image built FROM it. Build it from the repository root:
#
#   docker build -f packaging/runtime/Dockerfile -t markitect-runtime .
#
# A project builds its own image FROM this one and adds its toolchain. The
# host names the resulting image by its engine image ID.

FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY internal ./internal
COPY cmd/markitect-agentd ./cmd/markitect-agentd
COPY cmd/markitect ./cmd/markitect
ARG MARKITECT_VERSION=development
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=${MARKITECT_VERSION}" \
      -o /out/markitect-agentd ./cmd/markitect-agentd \
 && CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /out/markitect ./cmd/markitect

FROM node:22-bookworm-slim@sha256:c3de60bf2f9dd0ac6370e6117950ff62d6e339527e7472301c9c78a017978392
ARG CODEX_VERSION=0.162.0
RUN apt-get update \
 && apt-get install --yes --no-install-recommends ca-certificates git python3 \
 && rm -rf /var/lib/apt/lists/*
RUN npm install --global --no-fund --no-audit \
      "@openai/codex@${CODEX_VERSION}" \
 && npm cache clean --force
COPY --from=build /out/markitect-agentd /usr/local/bin/markitect-agentd
COPY --from=build /out/markitect /usr/local/bin/markitect

# Fresh named volumes mounted at these paths take over their ownership, so
# the non-root session user can write its private home and its clone.
RUN mkdir -p /home/agent /work && chown 1000:1000 /home/agent /work

# Record the pinned CLIs and the builds. The checks use a throwaway home, so
# no CLI state lands in the image, and fail the build when a CLI is not the
# pinned one or lacks a launch flag its adapter passes (Claude Code's two
# prompt-file flags are documented but not listed in --help).
ARG CLAUDE_CODE_VERSION=2.1.308
RUN npm install --global --no-fund --no-audit \
      "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
 && npm cache clean --force
RUN set -eu; \
    export HOME=/tmp/cli-check DISABLE_AUTOUPDATER=1 CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1; \
    mkdir -p "$HOME"; \
    claude=$(claude --version); \
    test "$claude" = "${CLAUDE_CODE_VERSION} (Claude Code)"; \
    help=$(claude --help); \
    for flag in --print --input-format --output-format --verbose --resume --model --effort \
        --permission-mode --permission-prompts --tools --disable-slash-commands --setting-sources --settings \
        --strict-mcp-config --mcp-config; do \
      printf '%s\n' "$help" | grep -qE -- "${flag}( |,|$)" || { echo "claude ${claude} lacks ${flag}" >&2; exit 1; }; \
    done; \
    codex=$(codex --version 2>/dev/null); \
    test "$codex" = "codex-cli ${CODEX_VERSION}"; \
    for command in "exec" "exec resume"; do \
      help=$(codex $command --help 2>/dev/null); \
      for flag in --json --disable --dangerously-bypass-approvals-and-sandbox --ignore-user-config --ignore-rules; do \
        printf '%s\n' "$help" | grep -qE -- "${flag}( |,|$)" || { echo "codex ${command} lacks ${flag}" >&2; exit 1; }; \
      done; \
    done; \
    rm -rf "$HOME"; \
    mkdir -p /opt/markitect; \
    printf 'claude=%s\ncodex=%s\nagentd=%s\nmarkitect=%s\n' \
      "$claude" "$codex" "$(markitect-agentd version)" "$(markitect version)" \
      > /opt/markitect/runtime.env

ENV HOME=/home/agent
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/markitect-agentd", "serve"]

# The evaluation sandbox: one image for every variant of the evaluation suite
# and for its egress gateway. Debian bookworm with Git, the Go, Node and Python
# versions of the Markitect runtime image (packaging/runtime/Dockerfile) and
# the same pinned Codex and Claude Code CLIs. Build it from this directory:
#
#   docker build -t markitect-eval-sandbox tools/evaluation/sandbox
#
# A series names the result by its engine image ID (sha256:...).

FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS go

# The runtime image's final stage: Node 22 and Debian bookworm's Python 3.
FROM node:22-bookworm-slim@sha256:c3de60bf2f9dd0ac6370e6117950ff62d6e339527e7472301c9c78a017978392
ARG CODEX_VERSION=0.162.0
ARG CLAUDE_CODE_VERSION=2.1.308
RUN apt-get update \
 && apt-get install --yes --no-install-recommends ca-certificates git python3 \
 && rm -rf /var/lib/apt/lists/*
COPY --from=go /usr/local/go /usr/local/go
RUN npm install --global --no-fund --no-audit \
      "@openai/codex@${CODEX_VERSION}" "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
 && npm cache clean --force
COPY egress.py /opt/evaluation/egress.py
COPY entry.sh /usr/local/bin/sandbox-entry
# Mount points of the suite's mounts, present on the read-only root.
RUN mkdir -p /task /out /state /variant /login && touch /login/credentials.json

# The network allows only the provider's endpoints: Go never fetches modules or
# toolchains, and the CLIs send no telemetry, error reports or update checks.
ENV PATH=/usr/local/go/bin:$PATH \
    GOPROXY=off GOTOOLCHAIN=local GOSUMDB=off \
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 DISABLE_AUTOUPDATER=1 \
    HOME=/tmp/home

# Fail the build when a CLI or toolchain is not the pinned one, and record the
# versions. The checks use a throwaway home, so no CLI state lands in the image.
RUN set -eu; \
    chmod 0755 /usr/local/bin/sandbox-entry; \
    export HOME=/tmp/version-check; \
    claude=$(claude --version); codex=$(codex --version); go=$(go version); \
    rm -rf /tmp/version-check; \
    test "$claude" = "${CLAUDE_CODE_VERSION} (Claude Code)"; \
    test "$codex" = "codex-cli ${CODEX_VERSION}"; \
    case "$go" in "go version go1.27.1 "*) ;; *) echo "unexpected $go" >&2; exit 1;; esac; \
    printf 'claude=%s\ncodex=%s\ngo=%s\npython=%s\ngit=%s\nnode=%s\n' \
      "$claude" "$codex" "$go" "$(python3 --version)" "$(git --version)" "$(node --version)" \
      > /opt/evaluation/sandbox.env

# A non-root user; the suite starts every container with a read-only root and
# a private /tmp, where HOME lives.
USER 1000:1000
WORKDIR /tmp
ENTRYPOINT ["/usr/local/bin/sandbox-entry"]

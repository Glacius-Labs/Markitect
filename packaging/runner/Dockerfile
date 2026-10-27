# Markitect Linux runner: the Apply host in one container (DEC-077,
# docs/design/linux-runner.md). Build it from the repository root and name the
# source revision, which becomes the binary's build identity
# (<version>+<revision>, the run configuration's productBuild):
#
#   docker build -f packaging/runner/Dockerfile \
#     --build-arg MARKITECT_REVISION="$(git rev-parse HEAD)" -t markitect-runner .
#
# Start it with `markitect apply --runner IMAGE_ID ...`, which mounts the
# checkout's .git directory and the logins read-only, the engine socket and
# the hand-back area.
# The image holds no credentials and no repository content.

FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/markitect ./cmd/markitect
COPY cmd/markitect-runner ./cmd/markitect-runner
COPY internal ./internal
ARG MARKITECT_REVISION=
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X github.com/Glacius-Labs/Markitect/internal/buildinfo.injectedRevision=${MARKITECT_REVISION}" \
      -o /out/markitect ./cmd/markitect \
 && CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w" \
      -o /out/markitect-runner ./cmd/markitect-runner

# The static Docker CLI. Apply drives the engine through the mounted socket;
# volume subpath mounts need CLI and engine 26 or newer.
FROM docker:29.8.2-cli@sha256:35cfcf893c20f2931f4909151b5407aaaeab711277de7ce8b8f521f095d54c32 AS docker

FROM debian:trixie-slim@sha256:62bd423aeee912639ca8ee681c2b762cb3e25137e0a92b20be455898a9f034ad
# Git only: the runner has no network (clone over the file transport, engine
# over its socket), so it needs no CA certificates.
RUN apt-get update \
 && apt-get install --yes --no-install-recommends git \
 && rm -rf /var/lib/apt/lists/*
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
COPY --from=build /out/markitect /out/markitect-runner /usr/local/bin/

# Mount targets: the checkout's Git directory (never its working tree), the
# invocation, logins, hand-back and state. A fresh state volume takes over the
# directory's ownership.
RUN mkdir -p /markitect/source.git /markitect/invocation /markitect/credentials \
      /markitect/output /markitect/state /markitect/work \
 && chmod 0700 /markitect/credentials /markitect/state /markitect/work

# Record what the image runs, for the operator and evidence. productBuild is
# the build the run configuration must assert.
ARG MARKITECT_REVISION=
RUN set -eu; \
    version=$(markitect version | sed -n 's/.*"version": *"\([^"]*\)".*/\1/p'); \
    build="$version"; \
    if [ -n "${MARKITECT_REVISION}" ]; then build="$version+${MARKITECT_REVISION}"; fi; \
    mkdir -p /opt/markitect; \
    printf 'productBuild=%s\nrevision=%s\ndocker=%s\ngit=%s\n' \
      "$build" "${MARKITECT_REVISION}" "$(docker --version)" "$(git --version)" \
      > /opt/markitect/runner.env
LABEL org.opencontainers.image.revision="${MARKITECT_REVISION}"

# The runner is trusted host code: with the engine socket it controls the
# engine anyway. It runs as root so that it can use the socket on every
# engine. Agents never run here; they run in the hardened sibling containers.
WORKDIR /markitect
ENTRYPOINT ["/usr/local/bin/markitect-runner"]

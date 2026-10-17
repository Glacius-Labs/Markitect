# Example project runtime image: the Markitect base image plus the Go
# toolchain, for projects such as the library example whose checks run
# `go test`. Build the base image first, then from the repository root:
#
#   docker build -f packaging/runtime/examples/go/Dockerfile -t markitect-runtime-go .
#
# Sessions and checks then use the same tools. The root filesystem stays
# read-only, so Go's caches live in the provider home and /tmp.

ARG BASE=markitect-runtime
FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS go

FROM ${BASE}
COPY --from=go /usr/local/go /usr/local/go
# Agent CLIs run commands in login shells, whose /etc/profile resets PATH;
# links on the default PATH keep the toolchain reachable there too.
USER root
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
USER 1000:1000
ENV PATH=/usr/local/go/bin:${PATH} \
    GOTOOLCHAIN=local \
    GOPATH=/home/agent/go \
    GOCACHE=/home/agent/.cache/go-build \
    GOFLAGS=-buildvcs=false

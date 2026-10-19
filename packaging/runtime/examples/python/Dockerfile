# Example project runtime image: the Markitect base image plus CPython 3.13,
# for projects such as Roombook and Readinglog whose checks run
# `python -B -m unittest`. Build the base image first, then from the
# repository root:
#
#   docker build -f packaging/runtime/examples/python/Dockerfile -t markitect-runtime-python .
#
# Sessions and checks then use the same interpreter. The root filesystem
# stays read-only, so nothing may require bytecode caches: the image sets
# PYTHONDONTWRITEBYTECODE and the projects call `python -B`.

ARG BASE=markitect-runtime
FROM python:3.13-slim-bookworm@sha256:a1165e272e578941b84abc79e4ab38a0305cd12803a5c4247979ac7655f4d641 AS python

FROM ${BASE}
# The base keeps Node and the Agent CLIs under /usr/local, so copy only what
# the interpreter needs: the executable, the standard library and the shared
# library the executable links against (rpath $ORIGIN/../lib).
COPY --from=python /usr/local/bin/python3.13 /usr/local/bin/python3.13
COPY --from=python /usr/local/lib/python3.13 /usr/local/lib/python3.13
COPY --from=python /usr/local/lib/libpython3.13.so.1.0 /usr/local/lib/libpython3.13.so.1.0
# Agent CLIs run commands in login shells, whose /etc/profile resets PATH;
# links on the default PATH keep the interpreter reachable there too. The
# base already carries the C libraries of the standard-library modules
# (OpenSSL, SQLite, zlib, bzip2, lzma, readline, libdb, time zones).
# Image observation checks every ancestor directory of a check executable and
# the engine walks each one's whole subtree, which is slow for /usr/local. A
# small regular wrapper in a small directory is the check executable.
USER root
RUN ln -s python3.13 /usr/local/bin/python3 \
 && ln -s python3.13 /usr/local/bin/python \
 && ln -s libpython3.13.so.1.0 /usr/local/lib/libpython3.13.so \
 && mkdir -p /opt/markitect/bin \
 && printf '#!/bin/sh\nexec /usr/local/bin/python3.13 "$@"\n' > /opt/markitect/bin/python \
 && chmod 755 /opt/markitect/bin/python
USER 1000:1000
ENV LANG=C.UTF-8 \
    PYTHONDONTWRITEBYTECODE=1

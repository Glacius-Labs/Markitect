#!/bin/sh
# The root is read-only and /tmp is a fresh tmpfs: create the home, then run the command.
set -e
mkdir -p "$HOME"
exec "$@"

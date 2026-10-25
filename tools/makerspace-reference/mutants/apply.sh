#!/bin/sh
# Usage: apply.sh FAMILY DEST
# Copies the step tree of FAMILY to DEST (which must not exist) and applies the mutant patch.
set -e
here=$(cd "$(dirname "$0")" && pwd)
family=$1
dest=$2
[ -n "$family" ] && [ -n "$dest" ] || { echo "usage: apply.sh FAMILY DEST" >&2; exit 2; }
[ ! -e "$dest" ] || { echo "$dest exists" >&2; exit 2; }
step=$(tr -d '\r\n' < "$here/$family/STEP")
cp -r "$here/../$step" "$dest"
(cd "$dest" && git apply --unsafe-paths -p1 "$here/$family/mutant.patch")

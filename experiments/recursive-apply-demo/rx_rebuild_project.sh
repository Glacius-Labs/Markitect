#!/usr/bin/env bash
# Create a case-1 target from an adopted Extract: the extracted model, the
# imported vocabulary and the lock, without any artifact or registry. The
# execution document keeps the Extract's layout exceptions and gets the library
# budget and the two documented checks (go build, go test). Their requiredBy is
# the first Definition Root registered realization/go.mod against, so the
# harness picks no Definition by its name. Apply then rebuilds the realization
# from the model alone; the oracle judges the result.
# Usage: rx_rebuild_project.sh ADOPTED_REPO TARGET
set -euo pipefail
ADOPTED="$1"
TARGET="$2"
DEMO="$(cd "$(dirname "$0")" && pwd)"
test ! -e "$TARGET"
mkdir -p "$TARGET/.markitect"
cp "$ADOPTED/.markitect/model.yaml" "$ADOPTED/.markitect/model-lock.yaml" "$TARGET/.markitect/"
cp -r "$ADOPTED/.markitect/model" "$ADOPTED/.markitect/dependencies" "$TARGET/.markitect/"
rm -f "$TARGET/.markitect/model/.gitkeep"
python3 - "$ADOPTED" "$TARGET" "$DEMO/library-execution.yaml" <<'PY'
import re
import sys
from pathlib import Path

adopted, target, template = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
extracted = (adopted / ".markitect/execution.yaml").read_text()
registry = (adopted / ".markitect/registry/registry.yaml").read_text()
# The layout exceptions block of the extracted execution document, verbatim.
m = re.search(r"^  exceptions:.*?(?=^\S|\Z)", extracted, re.S | re.M)
exceptions = m.group(0) if m else "  exceptions: []\n"
# The first Definition Root registered realization/go.mod against.
m = re.search(r"- path: realization/go\.mod\n(?:\s+\w+:.*\n)*?\s+realizes: \[([^\],]+)", registry) or \
    re.search(r"- path: realization/go\.mod\n(?:\s+\w+:.*\n)*?\s+realizes:\s*\n\s+- (\S+)", registry)
if not m:
    sys.exit("no go.mod registration in the root shard")
required = m.group(1).strip()
text = template.read_text()
text = re.sub(r"^  exceptions:.*?(?=^checks:)", exceptions, text, flags=re.S | re.M)
text = re.sub(r"requiredBy: \S+", "requiredBy: " + required, text)
(target / ".markitect/execution.yaml").write_text(text)
print("requiredBy", required)
PY
cd "$TARGET"
git init --quiet --initial-branch=main
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" add .
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" commit --quiet -m "Extracted model without artifacts (case 1)"
git rev-parse HEAD

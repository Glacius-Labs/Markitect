#!/usr/bin/env bash
# Create a case-1 target from an adopted Extract: the extracted model, the
# imported vocabulary and the lock, without any artifact or registry. The
# execution document keeps the Extract's ownership areas and gets the library
# budget and the two documented checks (go build, go test). Their requiredBy is
# the first Definition realization/go.mod is registered against, so the
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
# Every shard: the build file belongs to Root or, under an ownership area, to the project Package.
registry = "\n".join(p.read_text() for p in sorted((adopted / ".markitect/registry").rglob("registry.yaml")))
# The ownership block of the extracted execution document, verbatim.
m = re.search(r"^ownership:.*?(?=^\S|\Z)", extracted, re.S | re.M)
if not m:
    sys.exit("no ownership block in the extracted execution document")
ownership = m.group(0).rstrip("\n") + "\n"
# The first Definition the build file realization/go.mod is registered against.
m = re.search(r"- path: \"?realization/go\.mod\"?\n(?:\s+\w+:.*\n)*?\s+realizes: \[([^\],]+)", registry) or \
    re.search(r"- path: \"?realization/go\.mod\"?\n(?:\s+\w+:.*\n)*?\s+realizes:\s*\n\s+- (\S+)", registry)
if not m:
    sys.exit("no go.mod registration in any shard")
required = m.group(1).strip()
text = template.read_text()
text, n = re.subn(r"^ownership:.*?(?=^checks:)", lambda m: ownership, text, flags=re.S | re.M)
if n != 1:
    sys.exit("no ownership block before checks in the library execution document")
text = re.sub(r"requiredBy: \S+", "requiredBy: " + required, text)
(target / ".markitect/execution.yaml").write_text(text)
print("requiredBy", required)
PY
cd "$TARGET"
git init --quiet --initial-branch=main
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" add .
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" commit --quiet -m "Extracted model without artifacts (case 1)"
git rev-parse HEAD

#!/usr/bin/env bash
# The timed real demonstration: projection, Apply M1 -> A1, the one model
# change, Apply M2 -> A2. One hard wall-clock limit of 30 minutes covers all of
# it including stop and cleanup; it is enforced here, outside every prompt.
# Usage: ra_demo.sh ATTEMPT_DIR SRC_DIR
set -uo pipefail
ATTEMPT="$1"
SRC="$2"
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
T0=$(date +%s)
mkdir -p "$ATTEMPT/out"
{
  echo "start_utc=$(date -u -d @$T0 +%FT%TZ) epoch=$T0"
  echo "hard_limit_utc=$(date -u -d @$((T0 + 1800)) +%FT%TZ)"
} > "$ATTEMPT/timing.txt"
# The engine deadline leaves room for a graceful stop; timeout kills the whole
# process group at 27.5 minutes and again 45 seconds later; cleanup is bounded.
DEADLINE=$(date -u -d @$((T0 + 1560)) +%FT%TZ)
timeout --kill-after=45 1650 bash "$ATTEMPT/frozen/ra_inner.sh" "$ATTEMPT" "$SRC" "$SHA" "$DEADLINE"
code=$?
echo "inner_exit=$code elapsed=$(( $(date +%s) - T0 ))" >> "$ATTEMPT/timing.txt"
if [ "$code" = 124 ] || [ "$code" = 137 ]; then
  echo "outcome=INCOMPLETE hard limit reached" >> "$ATTEMPT/timing.txt"
fi
timeout 60 bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
echo "end_utc=$(date -u +%FT%TZ) total_seconds=$(( $(date +%s) - T0 ))" >> "$ATTEMPT/timing.txt"
cat "$ATTEMPT/timing.txt"

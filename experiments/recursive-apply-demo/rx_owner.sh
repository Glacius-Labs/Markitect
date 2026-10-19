#!/usr/bin/env bash
# Scripted owner for Extract scenarios: answers every question of the run's
# owner channel after a short delay and logs question and answer. A question
# about the loan period (14 or 21 days) gets the scenario's decision; any other
# question gets a neutral answer that settles nothing. Stops when STOP exists.
# Usage: rx_owner.sh MARKITECT RUN_CONFIG STATE_DIR LOG STOP
set -uo pipefail
MARKITECT="$1"
CONFIG="$2"
STATE="$3"
LOG="$4"
STOP="$5"
PERIOD="The intended loan period is 21 calendar days, as LENDING.md states. README.md and the implementation are outdated on this point."
NEUTRAL="I have no further decision on this. Describe what the artifacts support and keep anything they do not settle visible as an open point."
while [ ! -e "$STOP" ]; do
  for q in "$STATE"/questions/*.json; do
    [ -e "$q" ] || continue
    id=$(basename "$q" .json)
    [ -e "$STATE/answers/$id.txt" ] && continue
    grep -q "^$id " "$LOG" 2>/dev/null && continue
    sleep 20
    if grep -Eqi '21|14 ?(calendar )?days|loan period' "$q"; then answer="$PERIOD"; kind=period; else answer="$NEUTRAL"; kind=neutral; fi
    "$MARKITECT" answer "$id" --text "$answer" --run-config "$CONFIG" > /dev/null 2>&1
    echo "$id $kind exit=$? $(date -u +%FT%TZ)" >> "$LOG"
    cp "$q" "$(dirname "$LOG")/question-$id.json"
  done
  sleep 5
done

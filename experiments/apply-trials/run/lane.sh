#!/usr/bin/env bash
# Run trial cases one after another. Usage: lane.sh BUILD LANE_FILE
# Each LANE_FILE line: CASE ATTEMPT [FROM_ATTEMPT|-] [PREVIOUS_REALIZATION|-]; FROM_ATTEMPT names an
# earlier attempt whose repository is the start (--from-repo). Existing attempts are skipped. A frozen
# copy of run_case.sh runs each attempt, so editing the harness never breaks a running lane.
set -uo pipefail
BUILD="$1"; LANE="$2"
WORK="${TRIALS_WORK:-$HOME/mk-tr}"
HERE="$(cd "$(dirname "$0")" && pwd)"
while read -r CASE NAME FROM PREV; do
  case "$CASE" in ''|'#'*) continue ;; esac
  ATTEMPT="$WORK/$NAME"
  if [ -e "$ATTEMPT" ]; then echo "skip $NAME (exists)"; continue; fi
  FROMREPO=""; [ -n "${FROM:-}" ] && [ "$FROM" != "-" ] && FROMREPO="$WORK/$FROM/repo"
  PREVIOUS=""; [ -n "${PREV:-}" ] && [ "$PREV" != "-" ] && PREVIOUS="$PREV"
  if [ -n "$FROMREPO" ] && [ ! -d "$FROMREPO" ]; then echo "skip $NAME: $FROMREPO missing"; continue; fi
  cp "$HERE/run_case.sh" "$WORK/.run_case.$NAME.sh"
  cp "$HERE/patch_runconfig.py" "$WORK/patch_runconfig.py"
  echo "start $NAME $CASE $(date -u +%FT%TZ)"
  bash "$WORK/.run_case.$NAME.sh" "$BUILD" "$CASE" "$ATTEMPT" "$FROMREPO" "$PREVIOUS" > "$WORK/$NAME.log" 2>&1
  mv "$WORK/.run_case.$NAME.sh" "$ATTEMPT/run_case.frozen.sh" 2>/dev/null
  echo "done  $NAME $(tr '\n' ' ' < "$WORK/$NAME.log" | tail -c 300)"
done < "$LANE"
echo "lane finished $(date -u +%FT%TZ)"

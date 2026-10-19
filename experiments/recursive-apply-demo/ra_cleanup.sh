#!/usr/bin/env bash
# Stop every container of the demonstration and record what was still alive.
# Volumes holding retained work stay for evidence; remove them with docker volume rm.
# Usage: ra_cleanup.sh ATTEMPT_DIR
ATTEMPT="$1"
{
  echo "cleanup_at=$(date -u +%FT%TZ)"
  alive=$(docker ps -q --filter label=markitect.store=ra-library; docker ps -q --filter name=^markitect-ra; docker ps -q --filter label=markitect.check)
  alive=$(echo "$alive" | sort -u | grep . || true)
  echo "alive_containers=$(echo "$alive" | grep -c . || true)"
  if [ -n "$alive" ]; then
    docker ps --format '{{.Names}} {{.Status}}' --filter label=markitect.store=ra-library
    echo "$alive" | xargs -r docker rm -f > /dev/null 2>&1
  fi
  pkill -TERM -f "$ATTEMPT/markitect apply" 2>/dev/null && echo "terminated a remaining markitect process" || true
  echo "remaining_containers=$(docker ps -q --filter label=markitect.store=ra-library | grep -c . || true)"
} >> "$ATTEMPT/cleanup.txt" 2>&1
cat "$ATTEMPT/cleanup.txt"

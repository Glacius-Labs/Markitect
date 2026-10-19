#!/usr/bin/env bash
# Stop the Agent and workspace containers of this attempt's store label and
# record what was still alive. Containers of other stores (parallel sessions)
# are never touched; check containers end within their own timeout.
# Volumes holding retained work stay for evidence; remove them with docker volume rm.
# Usage: ra_cleanup.sh ATTEMPT_DIR
ATTEMPT="$1"
STORE=$(sed -n 's/^ *store: *//p' "$ATTEMPT/host/host.yaml" | head -1)
STORE="${STORE:-ra-library}"
{
  echo "cleanup_at=$(date -u +%FT%TZ) store=$STORE"
  alive=$(docker ps -q --filter "label=markitect.store=$STORE" | sort -u | grep . || true)
  echo "alive_containers=$(echo "$alive" | grep -c . || true)"
  if [ -n "$alive" ]; then
    docker ps --format '{{.Names}} {{.Status}}' --filter "label=markitect.store=$STORE"
    echo "$alive" | xargs -r docker rm -f > /dev/null 2>&1
  fi
  pkill -TERM -f "$ATTEMPT/markitect (apply|extract)" 2>/dev/null && echo "terminated a remaining markitect process" || true
  echo "remaining_containers=$(docker ps -q --filter "label=markitect.store=$STORE" | grep -c . || true)"
} >> "$ATTEMPT/cleanup.txt" 2>&1
cat "$ATTEMPT/cleanup.txt"

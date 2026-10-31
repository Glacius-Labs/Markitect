#!/usr/bin/env bash
# Starts Markitect Desktop's native app on a CI runner and checks that it
# serves the app as the native one, holds a window, has a later start open a
# window there, and, where a signal can reach it, quits cleanly.
#
#   scripts/desktop-native-smoke.sh APP CLI WORK
#
# APP is the built markitect-desktop, CLI the built markitect, WORK an empty
# folder. It uses the runner's own config folder, so run it on a fresh runner
# only. On Linux it starts its own X server and session bus.
set -euo pipefail

app="$1"
cli="$2"
work="$3"
mkdir -p "$work"

case "$(uname -s)" in
  Linux*) os=linux; config="${XDG_CONFIG_HOME:-$HOME/.config}/Markitect" ;;
  Darwin*) os=macos; config="$HOME/Library/Application Support/Markitect" ;;
  *) os=windows; config="$(cygpath -u "$APPDATA")/Markitect" ;;
esac
record="$config/desktop.json"
if [ -e "$record" ]; then
  echo "a Markitect already runs here ($record); use a fresh runner"
  exit 1
fi

fail() {
  echo "::error::$*"
  exit 1
}

# Whatever this run started ends with it, also when a check fails.
started=()
cleanup() {
  for p in "${started[@]}"; do
    if [ "$os" = windows ]; then taskkill //F //PID "$p" >/dev/null 2>&1 || true; else kill "$p" 2>/dev/null || true; fi
  done
}
trap cleanup EXIT

if [ "$os" = linux ]; then
  export DISPLAY=:99
  Xvfb :99 -screen 0 1600x1000x24 >"$work/xvfb.log" 2>&1 &
  started+=("$!")
  for _ in $(seq 1 30); do
    xdpyinfo >/dev/null 2>&1 && break
    sleep 1
  done
  xdpyinfo >/dev/null 2>&1 || fail "the X server did not start"
  eval "$(dbus-launch --sh-syntax)"
  started+=("$DBUS_SESSION_BUS_PID")
  # WebKitGTK renders without a GPU under Xvfb.
  export WEBKIT_DISABLE_COMPOSITING_MODE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 LIBGL_ALWAYS_SOFTWARE=1
fi

"$app" >"$work/app.log" 2>&1 &

for _ in $(seq 1 60); do
  [ -s "$record" ] && break
  sleep 1
done
[ -s "$record" ] || { cat "$work/app.log"; fail "no record after 60 s"; }
pid="$(sed -E 's/.*"pid":([0-9]+).*/\1/' "$record")"
started+=("$pid")
echo "the app runs as process $pid"

alive() {
  if [ "$os" = windows ]; then
    tasklist //FI "PID eq $1" //NH | grep -q "$1"
  else
    kill -0 "$1" 2>/dev/null
  fi
}

# The command line's later start finds the native service and prints a link.
out="$("$cli" desktop --no-open)"
echo "$out"
link="$(printf '%s\n' "$out" | sed -n -E 's/^open (http:[^ ]+) within.*/\1/p')"
[ -n "$link" ] || fail "the command line found no running Markitect"
base="${link%%/?t=*}"
jar="$work/cookies"
page="$(curl -fsS -c "$jar" -b "$jar" -L "$link")" || fail "the link did not open the app"
printf '%s' "$page" | grep -qi '<!doctype html>' || fail "the link did not serve the page"
app_info="$(curl -fsS -b "$jar" "$base/api/app")"
echo "$app_info"
printf '%s' "$app_info" | grep -q '"folderDialog":true' || fail "the service has no folder dialog, so it is not the native app's"

windows() {
  case "$os" in
    linux) xwininfo -root -tree | grep -c '"Markitect"' || true ;;
    windows) powershell -NoProfile -Command "(Get-Process -Id $pid).MainWindowTitle" | grep -c '^Markitect' || true ;;
    macos) swift "$work/windows.swift" "$pid" ;;
  esac
}
if [ "$os" = macos ]; then
  cat >"$work/windows.swift" <<'SWIFT'
import CoreGraphics
let pid = Int(CommandLine.arguments[1])!
let list = CGWindowListCopyWindowInfo(.optionAll, kCGNullWindowID) as? [[String: Any]] ?? []
print(list.filter { ($0[kCGWindowOwnerPID as String] as? Int) == pid && ($0[kCGWindowLayer as String] as? Int) == 0 }.count)
SWIFT
fi
sleep 5
alive "$pid" || { cat "$work/app.log"; fail "the app ended after it opened its window"; }
first="$(windows)"
echo "windows: $first"
[ "${first:-0}" -ge 1 ] || { cat "$work/app.log"; fail "the app shows no window"; }

# A later start of the app itself has the running one open a window and ends.
began=$(date +%s)
"$app" >"$work/second.log" 2>&1 || fail "the second start failed: $(cat "$work/second.log")"
echo "the second start ended after $(( $(date +%s) - began )) s"
alive "$pid" || fail "the first app ended with the second start"
sleep 3
if [ "$os" != windows ]; then
  # Windows reports one main window per process, so the count shows only
  # elsewhere.
  second="$(windows)"
  echo "windows: $second"
  [ "${second:-0}" -gt "$first" ] || fail "the second start opened no window"
fi

if [ "$os" = windows ]; then
  # No signal reaches a GUI program on Windows; its tray quits it.
  taskkill //F //PID "$pid" >/dev/null
  echo "smoke passed; quitting on Windows is the tray's"
  exit 0
fi
kill -TERM "$pid"
for _ in $(seq 1 20); do
  alive "$pid" || break
  sleep 1
done
alive "$pid" && fail "the app did not quit on SIGTERM"
[ -e "$record" ] && fail "the app left its record behind"
echo "smoke passed"

#!/bin/sh

########################################################

HOME="CHANGE_ME"
SERVICE_APP="$HOME/CHANGE_ME"
SERVICE_WEBROOT="$HOME/CHANGE_ME"
TOKEN="CHANGE_ME"

########################################################

if [ "$(cat)" != "$TOKEN" ]; then
  echo "Status: 401 Unauthorized"
  echo "Content-Type: text/plain; charset=utf-8"
  echo ""
  echo "Unauthorized"
  exit 0
fi

echo "Content-Type: text/plain; charset=utf-8"
echo ""

state=".$(basename "$0").state"
if ! mkdir "$state"; then
  echo "Another instance of this script is already running."
  echo "log:"
  cat "$state/log" 2>&1
  exit 0
fi
trap 'rm -rf "$state"; exit 0' EXIT

(

set -eux

export HOME
opt="$HOME/opt"
mkdir -p "$opt"

tool_path="$opt/webhosting-helper/bin"
mkdir -p "$tool_path"
PATH="$tool_path:$PATH"

if ! command -v webhosting-helper > /dev/null 2>&1; then
  url="https://github.com/win-t/go.winto.dev/releases/download/webhosting-helper%2Fv0.1.9/webhosting-helper"
  out="$opt/webhosting-helper/bin/webhosting-helper"
  curl -Lf --compressed -o "$out" "$url" || wget -qO "$out" "$url"
  chmod a+x "$out"
  webhosting-helper install-symlinks
fi

proxy-service-setup "$SERVICE_APP" "$SERVICE_WEBROOT"

SERVICE_DIR="${SERVICE_APP}.state"

dry_run=
case "$QUERY_STRING" in
  *dry_run=true*) dry_run=1 ;;
esac

if [ -z "$dry_run" ]; then
  env -i "HOME=$HOME" "PATH=$PATH" daemonize "$SERVICE_DIR" restart
  daemonize "$SERVICE_DIR" reopen
  echo "Service restarted successfully."
  sleep 1
fi

ps -eHo pid,ppid,time,rss,start,command
tail "$SERVICE_DIR/daemonize.state/log"

) 2>&1 | tee "$state/log"

#!/bin/bash
# Sample script to stop a single Tessera node using bash.
#
# Usage: ./stop-tessera.sample.sh <node-id>
# -----------------------------------------------------------------------------

if [ -z "$1" ]; then
  echo "err: no node-id provided"
  exit 1
fi

NODE_ID=$1

ps -eaf | grep tessera | grep -v grep | grep tessera-${NODE_ID}.json | tr -s " " " " | cut -d" " -f3 | xargs kill

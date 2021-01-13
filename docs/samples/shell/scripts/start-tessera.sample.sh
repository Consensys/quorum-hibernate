#!/bin/bash
# Sample script to start a single Tessera node using bash.
#
# Usage: ./start-tessera.sample.sh <node-id>
# -----------------------------------------------------------------------------

if [ -z "$1" ]; then
  echo "err: no node-id provided"
  exit 1
fi

NODE_ID=$1

java -Xms128M -Xmx128M -jar tessera-app.jar -configfile tessera-${NODE_ID}.json >> /logs/tessera-${NODE_ID}.log 2>&1 &

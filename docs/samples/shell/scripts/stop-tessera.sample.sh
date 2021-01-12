#!/bin/bash
# Sample script to stop a single Tessera node using bash.
#
# Stop scripts for Node Manager processes can be as simple or as complex as required.
# The script should only stop a single node (i.e. if multiple Tessera nodes are run on the same host,
# the stop script should only stop the intended node).
# This is just one instructive example.  The most appropriate stop script will depend on the specifics of the
# particular deployment.
#
# Usage: ./stop-tessera.sample.sh <node-id>
# -----------------------------------------------------------------------------

if [ -z "$1" ]; then
  echo "err: no node-id provided"
  exit 1
fi

NODE_ID=$1

ps -eaf | grep tessera | grep -v grep | grep tessera-${NODE_ID}.json | tr -s " " " " | cut -d" " -f3 | xargs kill

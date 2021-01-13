#!/bin/bash
# Sample script to stop a single GoQuorum node using bash.
#
# Usage: ./stop-goquorum.sample.sh <node-id>
# -----------------------------------------------------------------------------

if [ -z "$1" ]; then
  echo "err: no node-id provided"
  exit 1
fi

NODE_ID=$1

RPC_PORT="2200${NODE_ID}"

ps -eaf | grep geth | grep -v attach | grep -v grep | grep ${RPC_PORT} | tr -s " " " " | cut -d" " -f3 | xargs kill

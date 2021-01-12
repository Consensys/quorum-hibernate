#!/bin/bash
## Sample script to start a single GoQuorum node using bash.
##
## Usage: ./start-goquorum.sample.sh <node-id>
## -----------------------------------------------------------------------------

if [ -z "$1" ]; then
  echo "err: no node-id provided"
  exit 1
fi

NODE_ID=$1

PRIVATE_CONFIG=/tm.ipc geth \
  --datadir /qdata/dd${NODE_ID} --ws --wsapi admin,eth,debug --wsorigins=* --gcmode=archive --syncmode full \
  --nodiscover --allow-insecure-unlock --verbosity 5 --networkid 10 --raft --raftblocktime 50 --rpc --rpccorsdomain=* \
  --rpcvhosts=* --rpcaddr 0.0.0.0 \
  --rpcapi admin,eth,debug,miner,net,shh,txpool,personal,web3,quorum,raft,quorumPermission --emitcheckpoints \
  --unlock 0 --password passwords.txt --permissioned --wsport 2300${NODE_ID} --raftport 5040${NODE_ID} \
  --rpcport 2200${NODE_ID} --port 2100${NODE_ID} >>/logs/quorum-${NODE_ID} 2>&1 &

#!/bin/bash

nodenum=$1

let p=$nodenum-1
graphql=""
if [[ $nodenum -eq 1 ]]; then
   graphql=" --graphql --graphql.vhosts=* "
fi

# set the home directory for the Quorum blockchain node
BC_CLIENT_HOME_DIR=/tmp/quorum-examples/examples/7nodes
cd $BC_CLIENT_HOME_DIR


PRIVATE_CONFIG=qdata/c${nodenum}/tm.ipc geth --datadir qdata/dd${nodenum} --ws --wsapi admin,eth,debug --wsorigins=* --gcmode=archive --syncmode full --nodiscover --allow-insecure-unlock --verbosity 5 --networkid 10 --raft --raftblocktime 50 --rpc --rpccorsdomain=* --rpcvhosts=* --rpcaddr 0.0.0.0 --rpcapi admin,eth,debug,miner,net,shh,txpool,personal,web3,quorum,raft,quorumPermission --emitcheckpoints --unlock 0 --password passwords.txt --permissioned $graphql --wsport 2300${p} --raftport 5040${nodenum} --rpcport 2200${p} --port 2100${p} 2>> qdata/logs/${nodenum}.log &
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

PRIVATE_CONFIG=qdata/c${nodenum}/tm.ipc nohup geth --datadir qdata/dd${nodenum} --nodiscover --nousb --allow-insecure-unlock --networkid 10 --verbosity 3 --syncmode full --mine --minerthreads 1 --rpc --rpccorsdomain=* --rpcvhosts=* --rpcaddr 0.0.0.0 --rpcapi admin,eth,debug,miner,net,shh,txpool,personal,web3,quorum,quorumPermission,quorumExtension,clique --unlock 0 --password passwords.txt --wsport 2300${p}  --rpcport 2200${p} --port 2100${p} 2>>qdata/logs/${nodenum}.log &
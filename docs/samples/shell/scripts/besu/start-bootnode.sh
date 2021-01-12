#!/usr/bin/env bash
node=$1
# set the data directory path
DATA_PATH=/tmp/besu-network/clique-network/bdata/node-${node}/data
# set genesis file path
GENESIS=/tmp/besu-network/clique-network/cliqueGenesis.json

# create the log directory
mkdir -p $DATA_PATH/logs

# start the bootnode
besu --data-path=${DATA_PATH} --genesis-file=${GENESIS} --network-id 123 --p2p-port=3030${node} --rpc-http-enabled --rpc-http-api=ADMIN,ETH,NET,CLIQUE --host-allowlist="*" --rpc-http-cors-origins="all" --rpc-http-port=2200${node} &> ${DATA_PATH}/logs/${node}.log &
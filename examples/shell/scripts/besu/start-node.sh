#!/usr/bin/env bash
node=$1
# set the data directory path
DATA_PATH=/tmp/besu-network/clique-network/bdata/node-${node}/data
# set genesis file path
GENESIS=/tmp/besu-network/clique-network/cliqueGenesis.json

# create the log directory
mkdir -p $DATA_PATH/logs

## NOTE: replace the bootnode enode id in the below command before running the script !!!!
besu --data-path=${DATA_PATH} --genesis-file=${GENESIS} --bootnodes=<<enode id of the bootnode>> --network-id 123 --p2p-port=3030${node} --rpc-http-enabled --rpc-http-api=ETH,NET,CLIQUE --host-allowlist="*" --rpc-http-cors-origins="all" --rpc-http-port=2200${node} &> ${DATA_PATH}/logs/${node}.log &
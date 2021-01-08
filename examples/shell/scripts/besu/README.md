# Sample scripts

Sample scripts to start/stop a Besu node.

* `start-bootnode.sh`: Script to bring up the bootnode in the network
* `start-node.sh`: Script to bring up other nodes, referring to bootnode. 
* `stop-node.sh`: Script to stop a node

**NOTE:** Before using the start scripts, please edit the scripts to set the following:
* Set `DATA_PATH` and `GENESIS` appropriately in both `start-bootnode.sh` and `start-node.sh`

```bash
# set the data directory path
DATA_PATH=/tmp/besu-network/clique-network/bdata/node-${node}/data
# set genesis file path
GENESIS=/tmp/besu-network/clique-network/cliqueGenesis.json
```

* Set the enode id appropriately in `start-node.sh`

```bash
## NOTE: replace the bootnode enode id in the below command before running the script !!!!
besu --data-path=${DATA_PATH} --genesis-file=${GENESIS} --bootnodes=<<enode id of the bootnode>> --network-id 123 --p2p-port=3030${node} --rpc-http-enabled --rpc-http-api=ETH,NET,CLIQUE --host-allowlist="*" --rpc-http-cors-origins="all" --rpc-http-port=2200${node} &> ${DATA_PATH}/logs/${node}.log &
```
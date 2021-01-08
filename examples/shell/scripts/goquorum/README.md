# Sample scripts

Sample scripts to start/stop a GoQuorum node.

* `clique-start.sh`: Script to bring a GoQuroum node running with `clique` consensus
* `istanbul-start.sh`: Script to bring a GoQuroum node running with `istanbul` consensus
* `raft-start.sh`: Script to bring a GoQuroum node running with `raft` consensus
* `stop-node.sh`: Script to stop a node

**NOTE:** Set `BC_CLIENT_HOME_DIR` appropriately in all the start scripts

```bash
# set the home directory for the Quorum blockchain node
BC_CLIENT_HOME_DIR=/tmp/quorum-examples/examples/7nodes
cd $BC_CLIENT_HOME_DIR
```


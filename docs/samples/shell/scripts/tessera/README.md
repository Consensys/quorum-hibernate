# Sample scripts

Sample scripts to start/stop Tessera client.

* `start-tessera.sh`: Script to bring a GoQuroum node running with `clique` consensus
* `stop-tessera.sh`: Script to stop a node

**NOTE:** Before using the start scripts, please edit the `start-tessera.sh` to set the following:

* `BC_CLIENT_HOME_DIR`: Set to the 
* `JAVA_HOME`
* `TESSERA_JAR`

```bash
# set the home directory for the Quorum blockchain node
BC_CLIENT_HOME_DIR=/tmp/quorum-examples/examples/7nodes
cd $BC_CLIENT_HOME_DIR

# export Java home directory
export JAVA_HOME=/Library/Java/JavaVirtualMachines/openjdk-11.0.2.jdk/Contents/Home

# set Tessera jar path
TESSERA_JAR=/Users/tmp/tessera/tessera-app-0.11.1-SNAPSHOT-app.jar
```

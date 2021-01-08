#!/bin/bash
ptmnum=$1

# set the home directory for the Quorum blockchain node
BC_CLIENT_HOME_DIR=/tmp/quorum-examples/examples/7nodes
cd $BC_CLIENT_HOME_DIR

# export Java home directory
export JAVA_HOME=/Library/Java/JavaVirtualMachines/openjdk-11.0.2.jdk/Contents/Home

# set Tessera jar path
TESSERA_JAR=/Users/tmp/tessera/tessera-app-0.11.1-SNAPSHOT-app.jar

java -Xms128M -Xmx128M -jar ${TESSERA_JAR} -configfile qdata/c${ptmnum}/tessera-config-09-${ptmnum}.json >> qdata/logs/tessera${ptmnum}.log 2>&1 &

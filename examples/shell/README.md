# Running as host process

This contains the sample config files for node manager and peers in both `.toml` and `.json` form for bringing up node manager when the Blockchain Client and Privacy Manager are running as processes.

The node manager configuration requires start and stop scripts for the block chain client and Privacy Manager as described [here](./../../docs/CONFIG.md/#process). For sample start and stop scripts:

* If the Blockchain Client is GoQuorum, refer [this](scripts/goquorum) 
* If the Blockchain Client is Besu, refer [this](scripts/besu)
* If the Privacy Manager is Tessera, refer [this](scripts/tessera)
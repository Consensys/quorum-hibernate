# Running with Docker

This contains the sample config files for node manager and peers in both `.toml` and `.json` form for bringing up node manager when the blockchain client and privacy manager are running as processes.

The node manager configuration requires start and stop scripts for the block chain client and privacy manager as described [here](./../../README.md). For sample start and stop scripts:

* If the blockchain client is GoQuorum, refer [this](scripts/goquorum) 
* If the blockchain client is Besu, refer [this](scripts/besu)
* If the privacy manager is Tessera, refer [this](scripts/tessera)

**TODO: update the link for config once merged with doc PR**

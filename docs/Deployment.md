# Node Manager

## Supported Deployment Models
Node Manager must run in the same host where block chain client and privacy manager are running. Node Manager, block chain client and privacy manger can be run as host process or docker container. The supported combination is as given below.

| Node Manager  | Blockchain Client | Privacy Manager |
| :---: | :---: | :---: |
| Host process | Host process | Host process |
| Host process | Docker | Docker |
| Docker | Docker | Docker | 

## Up & Running

### Using Binary

#### Build

```bash
go install
```

#### Run

Ensure that `node-manager` is there in `$PATH`

```bash
node-manager --config path/to/config.json --verbosity 3
```

| Flag | Description |
| :---: | :--- |
| `--config` | Path to `.json` or `.toml` configuration file |
| `--verbosity` | Logging level (`0` = `ERROR`, `1` = `WARN`, `2` = `INFO`, `3` = `DEBUG`) |


### Using Docker

#### Build

```bash
docker build . -t node-manager
```
#### Run

Configuration files must be supplied to the Docker container. Refer to sample config files [config.toml](../examples/docker/nodemanager-config.sample.toml) and [nodemanager.toml](../examples/docker/peers.sample.toml)
```bash
docker run -p <port mapping> -v /var/run/docker.sock:/var/run/docker.sock --mount type=bind,source=<path to config>,target=/config.toml node-manager:latest

```

**Example**
```bash
docker run -p 8081:8081 -p 9091:9091 -p 9391:9391 -v /var/run/docker.sock:/var/run/docker.sock --mount type=bind,source=/usr/john/node1.toml,target=/config.toml --mount type=bind,source=/usr/john/nm1.toml,target=/nm1.toml node-manager:latest -config /config.toml
```
**Note:** `-v /var/run/docker.sock:/var/run/docker.sock` is required to start/stop blockchain client/privacy manager running as docker container.

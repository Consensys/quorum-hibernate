package config

type RPCServer struct {
	RPCAddr     string     `toml:"rpcAddress" json:"rpcAddress"`
	RPCCorsList []string   `toml:"rpcCorsList" json:"rpcCorsList"`
	RPCVHosts   []string   `toml:"rpcvHosts" json:"rpcvHosts"`
	TLSConfig   *ServerTLS `toml:"tlsConfig" json:"tlsConfig"`
}

func (c RPCServer) IsValid() error {
	if c.RPCAddr == "" {
		return newFieldErr("rpcAddress", isEmptyErr)
	}

	if c.TLSConfig != nil {
		if err := c.TLSConfig.IsValid(); err != nil {
			return newFieldErr("tlsConfig", err)
		}
	}
	return nil
}

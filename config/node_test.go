package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNodeManagerReader(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantImpl interface{}
	}{
		{
			name:     "toml",
			file:     "conf.toml",
			wantImpl: tomlNodeManagerReader{},
		},
		{
			name:     "json",
			file:     "conf.json",
			wantImpl: jsonNodeManagerReader{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewNodeManagerReader(tt.file)
			require.IsType(t, tt.wantImpl, r)
			require.NoError(t, err)
		})
	}
}

func TestNewNodeManagerReader_UnsupportedFileFormat(t *testing.T) {
	_, err := NewNodeManagerReader("conf.yaml")
	require.EqualError(t, err, "unsupported config file format")
}

func TestTomlNodeManagerReader_Read(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "toml",
			config: `
name = "node1"
upcheckPollingInterval = 1
peersConfigFile = "./test/shell/nm1.toml"
inactivityTime = 60
disableStrictMode = true

proxies = [
    { name = "geth-rpc", type = "http", proxyAddress = "localhost:9091", upstreamAddress = "http://localhost:22000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-graphql", type = "http", proxyAddress = "localhost:9191", upstreamAddress = "http://localhost:8547/graphql", proxyPaths = ["/graphql"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-ws", type = "ws", proxyAddress = "localhost:9291", upstreamAddress = "ws://localhost:23000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "tessera", type = "http", proxyAddress = "localhost:9391", upstreamAddress = "http://127.0.0.1:9001", proxyPaths = ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], readTimeout = 15, writeTimeout = 15 },
]

[server]
rpcAddress = "localhost:8081"
rpcCorsList = ["*"]
rpcvHosts = ["*"]

[blockchainClient]
type = "goquorum"
consensus = "raft"
rpcUrl = "http://localhost:22000"

[blockchainClient.process]
name = "bcclnt"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"]
upcheckConfig = { url = "http://localhost:22000", method = "POST", body = "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",returnType = "rpcresult"}

[privacyManager]
publicKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8="

[privacyManager.process]
name = "privman"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"]
upcheckConfig = { url = "http://localhost:9001/upcheck", method = "GET", body = "", returnType = "string", expected = "I'm up!"}
`,
		},
		{
			name: "json",
			config: `
{
	"name": "node1",
	"upcheckPollingInterval": 1,
	"peersConfigFile": "./test/shell/nm1.toml",
	"inactivityTime": 60,
	"disableStrictMode": true,
	"proxies": [
		{ "name": "geth-rpc", "type": "http", "proxyAddress": "localhost:9091", "upstreamAddress": "http://localhost:22000", "proxyPaths": ["/"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "geth-graphql", "type": "http", "proxyAddress": "localhost:9191", "upstreamAddress": "http://localhost:8547/graphql", "proxyPaths": ["/graphql"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "geth-ws", "type": "ws", "proxyAddress": "localhost:9291", "upstreamAddress": "ws://localhost:23000", "proxyPaths": ["/"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "tessera", "type": "http", "proxyAddress": "localhost:9391", "upstreamAddress": "http://127.0.0.1:9001", "proxyPaths": ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], "readTimeout": 15, "writeTimeout": 15 }
	],
	"server": {
		"rpcAddress": "localhost:8081",
		"rpcCorsList": ["*"],
		"rpcvHosts": ["*"]
	},
	"blockchainClient": {
		"type": "goquorum",
		"consensus": "raft",
		"rpcUrl": "http://localhost:22000",
		"process": {
			"name": "bcclnt",
			"controlType": "shell",
			"stopCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"],
			"startCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"],
			"upcheckConfig": { 
				"url": "http://localhost:22000", 
				"method": "POST", 
				"body": "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}", 
				"returnType": "rpcresult"
			}	
		}
	},
	"privacyManager": {
		"publicKey": "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
		"process": {
			"name": "privman",
			"controlType": "shell",
			"stopCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"],
			"startCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"],
			"upcheckConfig": { 
				"url": "http://localhost:9001/upcheck", 
				"method": "GET", 
				"body": "",
				"returnType": "string", 
				"expected": "I'm up!"
			}
		}
	}
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "qnmconfig")
			require.NoError(t, err)
			defer os.Remove(f.Name())

			_, err = f.Write([]byte(tt.config))
			require.NoError(t, err)

			var r NodeManagerReader
			if tt.name == "toml" {
				r = tomlNodeManagerReader{file: f.Name()}
			} else if tt.name == "json" {
				r = jsonNodeManagerReader{file: f.Name()}
			}
			got, err := r.Read()
			require.NoError(t, err)

			want := Basic{
				Name:                 "node1",
				UpchkPollingInterval: 1,
				PeersConfigFile:      "./test/shell/nm1.toml",
				InactivityTime:       60,
				DisableStrictMode:    true,
				Server: &RPCServer{
					RPCAddr:     "localhost:8081",
					RPCCorsList: []string{"*"},
					RPCVHosts:   []string{"*"},
				},
				BlockchainClient: &BlockchainClient{
					ClientType:   "goquorum",
					Consensus:    "raft",
					BcClntRpcUrl: "http://localhost:22000",
					//BcClntTLSConfig: nil,
					BcClntProcess: &Process{
						Name:         "bcclnt",
						ControlType:  "shell",
						ContainerId:  "",
						StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"},
						StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"},
						UpcheckCfg: &Upcheck{
							UpcheckUrl: "http://localhost:22000",
							Method:     "POST",
							Body:       "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",
							ReturnType: "rpcresult",
						},
					},
				},
				PrivacyManager: &PrivacyManager{
					PrivManKey: "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
					//PrivManTLSConfig: nil,
					PrivManProcess: &Process{
						Name:         "privman",
						ControlType:  "shell",
						ContainerId:  "",
						StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"},
						StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"},
						UpcheckCfg: &Upcheck{
							UpcheckUrl: "http://localhost:9001/upcheck",
							Method:     "GET",
							Body:       "",
							ReturnType: "string",
							Expected:   "I'm up!",
						},
					},
				},
				Proxies: []*Proxy{
					{
						Name:         "geth-rpc",
						Type:         "http",
						ProxyAddr:    "localhost:9091",
						UpstreamAddr: "http://localhost:22000",
						ProxyPaths:   []string{"/"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "geth-graphql",
						Type:         "http",
						ProxyAddr:    "localhost:9191",
						UpstreamAddr: "http://localhost:8547/graphql",
						ProxyPaths:   []string{"/graphql"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "geth-ws",
						Type:         "ws",
						ProxyAddr:    "localhost:9291",
						UpstreamAddr: "ws://localhost:23000",
						ProxyPaths:   []string{"/"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "tessera",
						Type:         "http",
						ProxyAddr:    "localhost:9391",
						UpstreamAddr: "http://127.0.0.1:9001",
						ProxyPaths:   []string{"/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
				},
			}

			require.Equal(t, want, got)
		})
	}
}

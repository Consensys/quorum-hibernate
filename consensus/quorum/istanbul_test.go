package quorum

import (
	"encoding/json"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIstanbulConsensus_ValidateShutdown_NonValidator_Valid(t *testing.T) {
	mockServer := startMockIstanbulServer(t, `{"result": false}`, "")
	defer mockServer.Close()

	istanbul := NewIstanbulConsensus(&types.NodeConfig{
		BasicConfig: &types.BasicConfig{
			BcClntRpcUrl: mockServer.URL,
		},
	})

	isConsensusNode, err := istanbul.ValidateShutdown()
	require.NoError(t, err)
	require.False(t, isConsensusNode)
}

func TestIstanbulConsensus_ValidateShutdown_Validator(t *testing.T) {
	var tests = []struct {
		name, istanbulIsValidatorResp, istanbulStatusResp string
		wantErrMsg                                        string
	}{
		{
			name:                    "noSealerActivity",
			istanbulIsValidatorResp: `{"result": true}`,
			istanbulStatusResp:      `{"result": {"numBlocks":0}}`,
			wantErrMsg:              "istanbul consensus check failed - no signers",
		},
		{
			name:                    "mintingNotStarted",
			istanbulIsValidatorResp: `{"result": true}`,
			istanbulStatusResp:      `{"result": {"numBlocks":0, "sealerActivity": {"somesigner":1}}}`,
			wantErrMsg:              "istanbul consensus check failed - block minting not started at network",
		},
		{
			name:                    "notEnoughPeers",
			istanbulIsValidatorResp: `{"result": true}`,
			istanbulStatusResp:      `{"result": {"numBlocks":10, "sealerActivity": {"minterone":10, "mintertwo":10, "minterthree":10}}}`,
			wantErrMsg:              "istanbul consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:0 numNodesDown:0",
		},
		{
			name:                    "notEnoughActivePeers",
			istanbulIsValidatorResp: `{"result": true}`,
			istanbulStatusResp:      `{"result": {"numBlocks":10, "sealerActivity": {"minterone":0, "mintertwo":10, "minterthree":10, "minterfour":10}}}`,
			wantErrMsg:              "istanbul consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:1 numNodesDown:1",
		},
		{
			name:                    "enoughActivePeers",
			istanbulIsValidatorResp: `{"result": true}`,
			istanbulStatusResp:      `{"result": {"numBlocks":10, "sealerActivity": {"minterone":10, "mintertwo":10, "minterthree":10, "minterfour":10}}}`,
			wantErrMsg:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := startMockIstanbulServer(t, tt.istanbulIsValidatorResp, tt.istanbulStatusResp)
			defer mockServer.Close()

			istanbul := NewIstanbulConsensus(&types.NodeConfig{
				BasicConfig: &types.BasicConfig{
					BcClntRpcUrl: mockServer.URL,
				},
			})

			isConsensusNode, err := istanbul.ValidateShutdown()
			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErrMsg)
			}
			require.True(t, isConsensusNode)
		})
	}
}

func startMockIstanbulServer(t *testing.T, istanbulIsValidatorResp, istanbulStatusResp string) *httptest.Server {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		type rpcRequest struct {
			Method string
		}

		rpcReq := rpcRequest{}

		err := json.NewDecoder(req.Body).Decode(&rpcReq)
		require.NoError(t, err)

		if rpcReq.Method == "istanbul_isValidator" {
			_, err := io.WriteString(w, istanbulIsValidatorResp)
			require.NoError(t, err)
		} else if rpcReq.Method == "istanbul_status" {
			_, err := io.WriteString(w, istanbulStatusResp)
			require.NoError(t, err)
		}
	})

	return httptest.NewServer(serverMux)
}

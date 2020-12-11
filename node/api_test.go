package node

import (
	"errors"
	"github.com/ConsenSysQuorum/node-manager/p2p"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/stretchr/testify/require"
)

func TestNewNodeRPCAPIs(t *testing.T) {
	service := &mockControllerApiService{}
	conf := &types.NodeConfig{}

	api := NewNodeRPCAPIs(service, conf)

	require.NotEmpty(t, api)
}

func TestNodeRPCAPIs_IsNodeUp_GetsStatusFromService(t *testing.T) {
	var (
		conf  = &types.NodeConfig{}
		param = new(string)
	)

	tests := []struct {
		name               string
		mockServiceResults map[string]interface{}
		want               NodeUpReply
	}{
		{
			name: "clientsAreUp",
			mockServiceResults: map[string]interface{}{
				"CheckClientUpStatus": true,
			},
			want: NodeUpReply{Status: true},
		},
		{
			name: "clientsAreDown",
			mockServiceResults: map[string]interface{}{
				"CheckClientUpStatus": false,
			},
			want: NodeUpReply{Status: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMockControllerApiService(tt.mockServiceResults)

			api := NewNodeRPCAPIs(service, conf)

			var got NodeUpReply

			err := api.IsNodeUp(nil, param, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, 1, service.callCount["CheckClientUpStatus"])
		})
	}
}

func TestNodeRPCAPIs_PrepareForPrivateTx(t *testing.T) {
	var (
		conf  = &types.NodeConfig{}
		param = new(string)
	)

	tests := []struct {
		name               string
		mockServiceResults map[string]interface{}
		want               PrivateTxPrepReply
		callCounts         map[string]int
		waitForGoroutine   bool
	}{
		{
			name: "ifNodeBusyDoNothing",
			mockServiceResults: map[string]interface{}{
				"IsNodeBusy": errors.New("someerror"),
			},
			want: PrivateTxPrepReply{Status: false},
			callCounts: map[string]int{
				"ResetInactiveSyncTime": 1,
				"IsNodeBusy":            1,
			},
		},
		{
			name: "ifClientsAlreadyUpPrepareThem",
			mockServiceResults: map[string]interface{}{
				"IsNodeBusy": nil,
				"IsClientUp": true,
			},
			want: PrivateTxPrepReply{Status: false},
			callCounts: map[string]int{
				"ResetInactiveSyncTime": 1,
				"IsNodeBusy":            1,
				"IsClientUp":            1,
				"PrepareClient":         1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMockControllerApiService(tt.mockServiceResults)

			api := NewNodeRPCAPIs(service, conf)

			var got PrivateTxPrepReply

			err := api.PrepareForPrivateTx(nil, param, &got)

			if tt.waitForGoroutine {
				time.Sleep(100 * time.Millisecond)
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.callCounts, service.callCount)
		})
	}
}

func TestTestNodeRPCAPIs_PrepareForPrivateTx_IfClientsDownPrepareInBackground(t *testing.T) {
	var (
		conf  = &types.NodeConfig{}
		param = new(string)
	)

	tests := []struct {
		name               string
		mockServiceResults map[string]interface{}
		want               PrivateTxPrepReply
		callCounts         map[string]int
	}{
		{
			name: "prepareClientSucceeds",
			mockServiceResults: map[string]interface{}{
				"IsNodeBusy":    nil,
				"IsClientUp":    false,
				"PrepareClient": true,
			},
			want: PrivateTxPrepReply{Status: false},
			callCounts: map[string]int{
				"ResetInactiveSyncTime": 1,
				"IsNodeBusy":            1,
				"IsClientUp":            1,
				"PrepareClient":         1,
			},
		},
		{
			name: "noErrIfPrepareClientFails",
			mockServiceResults: map[string]interface{}{
				"IsNodeBusy":    nil,
				"IsClientUp":    false,
				"PrepareClient": false,
			},
			want: PrivateTxPrepReply{Status: false},
			callCounts: map[string]int{
				"ResetInactiveSyncTime": 1,
				"IsNodeBusy":            1,
				"IsClientUp":            1,
				"PrepareClient":         1,
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMockControllerApiService(tt.mockServiceResults)

			api := NewNodeRPCAPIs(service, conf)

			var got PrivateTxPrepReply

			err := api.PrepareForPrivateTx(nil, param, &got)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NotContains(t, service.callCount, "PrepareClient")

			// wait for goroutine
			time.Sleep(100 * time.Millisecond)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.callCounts, service.callCount)
		})
	}
}

func TestNodeRPCAPIs_NodeStatus(t *testing.T) {
	var (
		conf = &types.NodeConfig{
			BasicConfig: &types.BasicConfig{
				InactivityTime: 50,
			},
		}
		param              = new(string)
		mockServiceResults = map[string]interface{}{
			"GetNodeStatus":          types.OK,
			"GetInactivityTimeCount": 40,
		}
		want = p2p.NodeStatusInfo{
			Status:            types.OK,
			InactiveTimeLimit: 50,
			InactiveTime:      40,
			TimeToShutdown:    10,
		}
		callCounts = map[string]int{
			"GetNodeStatus":          1,
			"GetInactivityTimeCount": 1,
		}
	)

	service := NewMockControllerApiService(mockServiceResults)

	api := NewNodeRPCAPIs(service, conf)

	var got p2p.NodeStatusInfo

	err := api.NodeStatus(nil, param, &got)

	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, callCounts, service.callCount)
}

func NewMockControllerApiService(results map[string]interface{}) *mockControllerApiService {
	return &mockControllerApiService{
		results:   results,
		callCount: make(map[string]int),
	}
}

type mockControllerApiService struct {
	results   map[string]interface{}
	callCount map[string]int
}

func (s *mockControllerApiService) CheckClientUpStatus(_ bool) bool {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return false
	}
	return s.results[getMethodName()].(bool)
}

func (s *mockControllerApiService) IsClientUp() bool {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return false
	}
	return s.results[getMethodName()].(bool)
}

func (s *mockControllerApiService) ResetInactiveSyncTime() {
	s.callCount[getMethodName()]++
	return
}

func (s *mockControllerApiService) IsNodeBusy() error {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return nil
	}
	return s.results[getMethodName()].(error)
}

func (s *mockControllerApiService) PrepareClient() bool {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return false
	}
	return s.results[getMethodName()].(bool)
}

func (s *mockControllerApiService) GetNodeStatus() types.NodeStatus {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return types.NodeStatus(0)
	}
	return s.results[getMethodName()].(types.NodeStatus)
}

func (s *mockControllerApiService) GetInactivityTimeCount() int {
	s.callCount[getMethodName()]++
	if s.results[getMethodName()] == nil {
		return 0
	}
	return s.results[getMethodName()].(int)
}

func getMethodName() string {
	pc, _, _, _ := runtime.Caller(1)
	nameFull := runtime.FuncForPC(pc).Name()
	nameEnd := filepath.Ext(nameFull)
	return strings.TrimPrefix(nameEnd, ".")
}

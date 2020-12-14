package node

import (
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/process"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNodeControl_CheckClientUpStatus_IfCachedStatusIsDownThenOnlyUpdateCacheWhenForced(t *testing.T) {
	var (
		initialClientStatus = core.Down
	)

	tests := []struct {
		name                 string
		bcClient             process.Process
		pmClient             process.Process
		forceConnectToClient bool
		want                 bool
		wantClientStatus     core.ClientStatus
	}{
		{
			name:                 "returnsCacheWithoutUpdating",
			bcClient:             &mockUpProcess{}, // if status was updated it would return as up
			pmClient:             &mockUpProcess{}, // if status was updated it would return as up
			forceConnectToClient: false,
			want:                 false,
			wantClientStatus:     core.Down,
		},
		{
			name:                 "forceUpdate_BcUpPmUp",
			bcClient:             &mockUpProcess{},
			pmClient:             &mockUpProcess{},
			forceConnectToClient: true,
			want:                 true,
			wantClientStatus:     core.Up,
		},
		{
			name:                 "forceUpdate_BcDownPmDown",
			bcClient:             &mockDownProcess{},
			pmClient:             &mockDownProcess{},
			forceConnectToClient: true,
			want:                 false,
			wantClientStatus:     core.Down,
		},
		{
			name:                 "forceUpdate_BcUpPmDown",
			bcClient:             &mockUpProcess{},
			pmClient:             &mockDownProcess{},
			forceConnectToClient: true,
			want:                 false,
			wantClientStatus:     core.Down,
		},
		{
			name:                 "forceUpdate_BcDownPmUp",
			bcClient:             &mockDownProcess{},
			pmClient:             &mockUpProcess{},
			forceConnectToClient: true,
			want:                 false,
			wantClientStatus:     core.Down,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NodeControl{
				clientStatus:  initialClientStatus,
				bcclntProcess: tt.bcClient,
				pmclntProcess: tt.pmClient,
				withPrivMan:   true,
			}

			got := n.CheckClientUpStatus(tt.forceConnectToClient)

			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantClientStatus, n.ClientStatus())
		})
	}
}

func TestNodeControl_CheckClientUpStatus_IfCachedStatusIsUpThenAlwaysUpdateCache(t *testing.T) {
	var (
		initialClientStatus  = core.Up
		forceConnectToClient = false
	)

	tests := []struct {
		name             string
		bcClient         process.Process
		pmClient         process.Process
		want             bool
		wantClientStatus core.ClientStatus
	}{
		{
			name:             "updates_BcUpPmUp",
			bcClient:         &mockUpProcess{},
			pmClient:         &mockUpProcess{},
			want:             true,
			wantClientStatus: core.Up,
		},
		{
			name:             "updates_BcDownPmDown",
			bcClient:         &mockDownProcess{},
			pmClient:         &mockDownProcess{},
			want:             false,
			wantClientStatus: core.Down,
		},
		{
			name:             "updates_BcUpPmDown",
			bcClient:         &mockUpProcess{},
			pmClient:         &mockDownProcess{},
			want:             false,
			wantClientStatus: core.Down,
		},
		{
			name:             "updates_BcDownPmUp",
			bcClient:         &mockDownProcess{},
			pmClient:         &mockUpProcess{},
			want:             false,
			wantClientStatus: core.Down,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NodeControl{
				clientStatus:  initialClientStatus,
				bcclntProcess: tt.bcClient,
				pmclntProcess: tt.pmClient,
				withPrivMan:   true,
			}

			got := n.CheckClientUpStatus(forceConnectToClient)

			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantClientStatus, n.ClientStatus())
		})
	}
}

type mockUpProcess struct{}

func (p *mockUpProcess) Start() error {
	panic("implement me")
}

func (p *mockUpProcess) Stop() error {
	panic("implement me")
}

func (p *mockUpProcess) UpdateStatus() bool {
	return true
}

func (p *mockUpProcess) Status() bool {
	return true
}

type mockDownProcess struct{}

func (p *mockDownProcess) Start() error {
	panic("implement me")
}

func (p *mockDownProcess) Stop() error {
	panic("implement me")
}

func (p *mockDownProcess) UpdateStatus() bool {
	return false
}

func (p *mockDownProcess) Status() bool {
	return false
}

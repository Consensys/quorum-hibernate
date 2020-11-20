package node

import (
	"errors"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/qnm"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type NodeRPCAPIs struct {
	qn *QuorumNodeControl
}

type NodeUpReply struct {
	Status bool
}

type PrivateTxPrepReply struct {
	Status bool
}

func NewNodeRPCAPIs(qn *QuorumNodeControl) *NodeRPCAPIs {
	return &NodeRPCAPIs{qn: qn}
}

func (n *NodeRPCAPIs) IsNodeUp(req *http.Request, from *string, reply *NodeUpReply) error {
	log.Debug("IsNodeUp - rpc call isNodeUp", "from", *from)
	ok := false
	if !n.qn.IsNodeUp() {
		log.Debug("IsNodeUp - is node up failed")
		ok = false
		return errors.New("IsNodeUp - is node up failed")
	}
	*reply = NodeUpReply{
		ok,
	}
	return nil
}

func (n *NodeRPCAPIs) PrepareForPrivateTx(req *http.Request, from *string, reply *PrivateTxPrepReply) error {
	log.Debug("PrepareForPrivateTx - rpc call - request received to prepare node", "from", *from)
	n.qn.ResetInactiveTime()
	status := n.qn.PrepareNode()
	*reply = PrivateTxPrepReply{Status: status}
	log.Info("PrepareForPrivateTx - rpc call - request processed to prepare node", "from", *from, "status", status)
	return nil
}

func (n *NodeRPCAPIs) NodeStatus(req *http.Request, from *string, reply *qnm.NodeStatusInfo) error {
	status := n.qn.GetNodeStatus()
	inactiveTimeLimit := n.qn.config.BasicConfig.InactivityTime
	curInactiveTimeCount := n.qn.im.GetInactivityTimeCount()
	*reply = qnm.NodeStatusInfo{Status: status, InactiveTimeLimit: inactiveTimeLimit, InactiveTime: curInactiveTimeCount, TimeToShutdown: inactiveTimeLimit - curInactiveTimeCount}
	log.Info("NodeStatus - rpc call", "from", *from, "status", status)
	return nil
}

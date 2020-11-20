package node

import (
	"errors"
	"github.com/ConsenSysQuorum/node-manager/qnm"
	"net/http"

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
	log.Info("rpc call isNodeUp", "from", *from)
	ok := false
	if !n.qn.IsNodeUp() {
		log.Info("is node up failed")
		ok = false
		return errors.New("is node up failed")
	}
	*reply = NodeUpReply{
		ok,
	}
	return nil
}

func (n *NodeRPCAPIs) PrepareForPrivateTx(req *http.Request, from *string, reply *PrivateTxPrepReply) error {
	log.Info("rpc call PrepareForPrivateTx - request received to prepare node", "from", *from)
	n.qn.ResetInactiveTime()
	status := n.qn.PrepareNode()
	*reply = PrivateTxPrepReply{Status: status}
	log.Info("rpc call PrepareForPrivateTx - request processed to prepare node", "from", *from, "status", status)
	return nil
}

func (n *NodeRPCAPIs) NodeStatus(req *http.Request, from *string, reply *qnm.NodeStatusInfo) error {
	status := n.qn.GetNodeStatus()
	inactiveTimeLimit := n.qn.config.BasicConfig.InactivityTime
	curInactiveTimeCount := n.qn.im.GetInactivityTimeCount()
	*reply = qnm.NodeStatusInfo{Status: status, InactiveTimeLimit: inactiveTimeLimit, InactiveTime: curInactiveTimeCount, TimeToShutdown: inactiveTimeLimit - curInactiveTimeCount}
	log.Info("rpc call NodeStatus", "from", *from, "status", status)
	return nil
}

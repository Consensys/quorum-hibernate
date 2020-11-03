package node

import (
	"github.com/ConsenSysQuorum/node-manager/log"
	"net/http"
)

type NodeRPCAPIs struct {
	qn *QuorumNode
}

type NodeUpReply struct {
	Status bool
}

type PrivateTxPrepReply struct {
	Status bool
}

func NewNodeRPCAPIs(qn *QuorumNode) *NodeRPCAPIs {
	return &NodeRPCAPIs{qn: qn}
}

func (n *NodeRPCAPIs) IsNodeUp(req *http.Request, from *string, reply *NodeUpReply) error {
	log.Info("rpc call isNodeUp", "from", *from)
	ok := false
	var err error
	if ok, err = n.qn.PingNodeToCheckIfItIsUp(); err != nil {
		log.Info("is node up failed", "err", err)
		ok = false
		return err
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

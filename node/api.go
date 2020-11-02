package node

import (
	"fmt"
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
	if ok, err = n.qn.IsNodeUp(); err != nil {
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
	log.Info("rpc call PrepareForPrivateTx", "from", *from)
	ok := false
	var err error
	n.qn.ResetInactiveTime()
	if ok, err = n.qn.IsNodeUp(); err == nil {
		if !ok {
			if err := n.qn.StartNode(false, false); err == nil {
				log.Info("node started successfully and ready for private tx")
				*reply = PrivateTxPrepReply{true}
				return nil
			} else {
				log.Info("node failed to started succesfully, not ready for private tx")
				return fmt.Errorf("node failed to start %v", err)
			}
		} else {
			log.Info("node is up and ready for private tx")
			*reply = PrivateTxPrepReply{true}
			return nil
		}
	} else {
		log.Info("is node up failed", "err", err)
		if err := n.qn.StartNode(false, false); err == nil {
			log.Info("node started successfully and ready for private tx")
			*reply = PrivateTxPrepReply{true}
			return nil
		} else {
			log.Info("node failed to started succesfully, not ready for private tx")
			return fmt.Errorf("node failed to start %v", err)
		}
	}
}

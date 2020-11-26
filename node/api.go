package node

import (
	"errors"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core/types"

	"github.com/ConsenSysQuorum/node-manager/qnm"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type NodeRPCAPIs struct {
	qn *NodeControl
}

type NodeUpReply struct {
	Status bool
}

type PrivateTxPrepReply struct {
	Status bool
}

func NewNodeRPCAPIs(qn *NodeControl) *NodeRPCAPIs {
	return &NodeRPCAPIs{qn: qn}
}

// IsNodeUp checks if the node is up and returns the node's up status
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

// PrepareForPrivateTx prepares this node for handling private transaction.
// it returns status as true if preparing the node is successful else it returns status as false
func (n *NodeRPCAPIs) PrepareForPrivateTx(req *http.Request, from *string, reply *PrivateTxPrepReply) error {
	log.Debug("PrepareForPrivateTx - rpc call - request received to prepare node", "from", *from)
	n.qn.ResetInactiveTime()
	var status bool
	if err := n.qn.IsNodeBusy(); err != nil {
		*reply = PrivateTxPrepReply{Status: false}
	} else {
		if n.qn.nodeStatus == types.Down {
			// send the response immediately and run prepare node in the background
			*reply = PrivateTxPrepReply{Status: false}
			go func() {
				log.Info("PrepareForPrivateTx - rpc call - prepareNode triggered")
				s := n.qn.PrepareNode()
				log.Info("PrepareForPrivateTx - rpc call - prepareNode triggered completed", "status", s)
			}()
		} else {
			status = n.qn.PrepareNode()
			*reply = PrivateTxPrepReply{Status: status}
		}
	}
	log.Info("PrepareForPrivateTx - rpc call - request processed to prepare node", "from", *from, "status", status)
	return nil
}

// NodeStatus returns current status of this node
func (n *NodeRPCAPIs) NodeStatus(req *http.Request, from *string, reply *qnm.NodeStatusInfo) error {
	status := n.qn.GetNodeStatus()
	inactiveTimeLimit := n.qn.config.BasicConfig.InactivityTime
	curInactiveTimeCount := n.qn.im.GetInactivityTimeCount()
	*reply = qnm.NodeStatusInfo{Status: status, InactiveTimeLimit: inactiveTimeLimit, InactiveTime: curInactiveTimeCount, TimeToShutdown: inactiveTimeLimit - curInactiveTimeCount}
	log.Info("NodeStatus - rpc call", "from", *from, "status", status)
	return nil
}

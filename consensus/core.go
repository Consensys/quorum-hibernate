package consensus

// Consensus is an interface for different consensus like raft, istanbul and clique.
//
// ValidateShutdown should be called before proceeding to shutdown node for decision making
//
// ValidateShutdown should check whether node can be shut down or not based on the
// status of other nodes in the cluster. It should make rpc calls to blockchain client's consensus specific
// APIS to decide that.
//
// For example, raft should call raft_cluster and raft_role APIs to decide whether node can be shutdown or no

type Consensus interface {
	// ValidateShutdown return mil if node is good to shutdown else returns error
	ValidateShutdown() (consensusNode bool, err error)
}

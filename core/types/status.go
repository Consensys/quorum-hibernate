package types

// NodeStatus indicates the combined status of both blockchain client and privacy manager
type NodeStatus uint8

const (
	ShutdownInprogress      NodeStatus = iota // indicates that shutdown process has started
	StartupInitiated                          // indicates start up of blockchain client and privacy manager has been initiated
	StartupInprogress                         // indicates start up of blockchain client and privacy manager is in progress
	Up                                        // indicates that both blockchain client and privacy manager are up
	Down                                      // indicates that both blockchain client and privacy manager are down
	WaitingPeerConfirmation                   // indicates the node has been inactive and awaiting peer confirmation to start shutdown
)

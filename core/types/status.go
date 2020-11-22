package types

// NodeStatus indicates the combined status of both geth and tessera
type NodeStatus uint8

const (
	ShutdownInitiated  NodeStatus = iota // indicates that shutdown initiated after confirming other qnms are not shuttingdown
	ShutdownInprogress                   // indicates that shutdown process has started
	StartupInitiated                     // indicates start up of geth and tessera has been initiated
	StartupInprogress                    // indicates start up of geth and tessera is in progress
	Up                                   // indicates that both geth and tessera are up
	Down                                 // indicates that both geth and tessera are down
)

package core

// ClientStatus indicates the combined status of both blockchain client and privacy manager processes
type ClientStatus uint8

const (
	_    ClientStatus = iota
	Up                // indicates both blockchain client and privacy manager are up
	Down              // indicates that both blockchain client and privacy manager are down
)

// NodeStatus indicates the status of node manager
type NodeStatus uint8

const (
	_                  NodeStatus = iota // indicates that node manager is shutting down both blockchain client and privacy manager
	ShutdownInprogress                   // indicates that node manager is shutting down both blockchain client and privacy manager
	StartupInprogress                    // indicates that node manager is starting up both blockchain client and privacy manager
	ConsensusWait
	OK // default status of node manager when its not doing anything
)
package types

type NodeStatus uint8

const (
	ShutdownInitiated NodeStatus = iota
	ShutdownInprogress
	StartupInitiated
	StartupInprogress
	Up
	Down
)

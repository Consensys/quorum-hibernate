package node

type Consensus interface {
	ValidateShutdown() error
}

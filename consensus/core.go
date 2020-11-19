package consensus

type Consensus interface {
	ValidateShutdown() error
}

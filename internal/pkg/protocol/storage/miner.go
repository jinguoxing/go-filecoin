package storage

import (
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("/fil/storage")

// Miner is
type Miner struct {
}

// NewMiner is
func NewMiner() (*Miner, error) {
	sm := &Miner{}

	panic("replace this with storage market in go-fil-markets")

	return sm, nil
}

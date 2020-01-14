package types

import (
	"github.com/filecoin-project/go-filecoin/internal/pkg/encoding"
)

func init() {
	encoding.RegisterIpldCborType(SectorPreCommitInfo{})
}

type SectorPreCommitInfo struct {
	SectorNumber Uint64

	CommR     []byte
	SealEpoch Uint64
	DealIDs   []Uint64
}

type SectorProveCommitInfo struct {
	Proof    []byte
	SectorID Uint64
	DealIDs  []Uint64
}

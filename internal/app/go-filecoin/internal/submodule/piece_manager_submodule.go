package submodule

import (
	"context"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/ipfs/go-datastore"
	"github.com/filecoin-project/go-storage-miner"

	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
)

// PieceManagerSubmodule enhances the `Node` with sector storage capabilities.
type PieceManagerSubmodule struct {
	// PieceManager is used by the miner to fill and seal sectors.
	PieceManager piecemanager.PieceManager
}

// NewPieceManagerSubmodule creates a new sector builder submodule.
func NewPieceManagerSubmodule(ctx context.Context, workerAddr, minerAddr address.Address, n StorageMinerNodeAPI, ds datastore.Batching, s *sectorbuilder.SectorBuilder) (PieceManagerSubmodule, error) {
	x := NewStorageMinerNode(n, workerAddr, minerAddr)
	sectorBuilder, err := storage.NewMiner(x, ds, s)
	if err != nil {
		return PieceManagerSubmodule{}, err
	}

	// TODO: write the software
	panic(sectorBuilder)

	return PieceManagerSubmodule{
		// sectorBuilder: nil,
	}, nil
}

package submodule

import (
	"context"

	"github.com/filecoin-project/go-filecoin/internal/pkg/postgenerator"

	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/msg"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-datastore"

	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
)

// StorageMiningSubmodule enhances the `Node` with storage mining capabilities.
type StorageMiningSubmodule struct {
	// StorageMining is used by the miner to fill and seal sectors.
	PieceManager piecemanager.PieceManager

	// PoStGenerator generates election PoSts
	PoStGenerator postgenerator.PoStGenerator
}

// NewStorageMiningSubmodule creates a new storage mining submodule.
func NewStorageMiningSubmodule(ctx context.Context, minerAddr, workerAddr address.Address, ds datastore.Batching, s sectorbuilder.Interface, c *ChainSubmodule, m *MessagingSubmodule, mw *msg.Waiter, w *WalletSubmodule) (StorageMiningSubmodule, error) {
	storageMiner, err := storage.NewMiner(NewStorageMinerNode(minerAddr, workerAddr, c, m, mw, w), ds, s)
	if err != nil {
		return StorageMiningSubmodule{}, err
	}

	return StorageMiningSubmodule{
		PieceManager:  piecemanager.NewChainBackedPieceManager(storageMiner, s, c.State),
		PoStGenerator: postgenerator.NewSectorBuilderBackedPoStGenerator(s),
	}, nil
}

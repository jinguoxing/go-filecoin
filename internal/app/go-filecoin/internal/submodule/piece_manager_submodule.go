package submodule

import (
	"context"
	"io"

	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/cst"

	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/msg"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
)

// PieceManagerSubmodule enhances the `Node` with piece management capabilities.
type PieceManagerSubmodule struct {
	// PieceManager is used by the miner to fill and seal sectors.
	PieceManager piecemanager.PieceManager
}

// NewPieceManagerSubmodule creates a new piece management submodule.
func NewPieceManagerSubmodule(ctx context.Context, minerAddr, workerAddr address.Address, ds datastore.Batching, s *sectorbuilder.SectorBuilder, c *ChainSubmodule, m *MessagingSubmodule, mw *msg.Waiter, w *WalletSubmodule) (PieceManagerSubmodule, error) {
	storageMiner, err := storage.NewMiner(NewStorageMinerNode(minerAddr, workerAddr, c, m, mw, w), ds, s)
	if err != nil {
		return PieceManagerSubmodule{}, err
	}

	return PieceManagerSubmodule{
		PieceManager: NewChainBackedPieceManager(storageMiner, s, c.State),
	}, nil
}

type ChainBackedPieceManager struct {
	miner   *storage.Miner
	builder *sectorbuilder.SectorBuilder
	chain   *cst.ChainStateReadWriter
}

func NewChainBackedPieceManager(m *storage.Miner, b *sectorbuilder.SectorBuilder, csrw *cst.ChainStateReadWriter) *ChainBackedPieceManager {
	return &ChainBackedPieceManager{
		miner:   m,
		builder: b,
		chain:   csrw,
	}
}

func (s *ChainBackedPieceManager) SealPieceIntoNewSector(ctx context.Context, dealID uint64, pieceSize uint64, pieceReader io.Reader) error {
	sectorID, offset, err := s.miner.AllocatePiece(pieceSize)
	if err != nil {
		return errors.Wrap(err, "failed to acquire sector id from storage miner")
	}

	if offset != 0 {
		return errors.New("`SealPieceIntoNewSector` assumes that the piece is always allocated into a newly-provisioned sector")
	}

	err = s.miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, dealID)
	if err != nil {
		return errors.Wrap(err, "storage miner `SealPiece` produced an error")
	}

	return nil
}

func (s *ChainBackedPieceManager) UnsealSector(sectorId uint64) (io.Reader, error) {
	s.builder.ReadPieceFromSealedSector(sectorId, 0, s.builder.SectorSize())
}

func (s *ChainBackedPieceManager) LocatePieceWithinSector(dealID uint64) (sectorID uint64, offset uint64, length uint64, err error) {
	panic("implement me")
}

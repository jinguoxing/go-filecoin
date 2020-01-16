package piecemanager

import (
	"context"
	"io"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-storage-miner"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/cst"
)

type ChainBackedPieceManager struct {
	miner   *storage.Miner
	builder sectorbuilder.Interface
	chain   *cst.ChainStateReadWriter
}

func NewChainBackedPieceManager(m *storage.Miner, b sectorbuilder.Interface, csrw *cst.ChainStateReadWriter) *ChainBackedPieceManager {
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

func (s *ChainBackedPieceManager) UnsealSector(ctx context.Context, sectorId uint64) (io.ReadCloser, error) {
	panic("use storage miner state to acquire ticket used when encoding sector")

	return s.builder.ReadPieceFromSealedSector(sectorId, 0, s.builder.SectorSize(), nil, nil)
}

func (s *ChainBackedPieceManager) LocatePieceForDealWithinSector(ctx context.Context, dealID uint64) (sectorID uint64, offset uint64, length uint64, err error) {
	panic("use storage market and storage miner actor state to locate piece within a sector")

	return 0, 0, 0, nil
}

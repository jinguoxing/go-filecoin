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

func (s *ChainBackedPieceManager) UnsealSector(ctx context.Context, sectorId uint64) (io.ReadCloser, error) {
	// get SectorPreCommitInfo from the miner actor's state with id = sectorId
	// get the
	var ticket []byte
	var commD []byte

	panic("need to acquire ticket and commD from actor state via state machine")

	return s.builder.ReadPieceFromSealedSector(sectorId, 0, s.builder.SectorSize(), ticket, commD)
}

func (s *ChainBackedPieceManager) LocatePieceWithinSector(ctx context.Context, dealID uint64) (sectorID uint64, offset uint64, length uint64, err error) {
	// load all pre-commit infos for a storage miner actor
	// find the pre-commit info whose []dealID contains the provided dealID
	//
	// acquire from storage miner actor a SealPreCommitInfo by dealID
	// for each deal id in SealPreCommitInfo.DealIDs
	//   get OnChainDeal info from storage market actor
	//   map those OnChainDeal thingies to piece sizes so that we can compute
	//      the offset and length

	panic("implement me")

	return 0, 0, 0, nil
}

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
	// - let ID = `sectorId`
	// - get slice of pre-committed sector info PSIS from storage miner actor storage
	// - let PSI = nil
	// - for each p in PSIS: if p.SectorNumber.ID == ID then { PSI = p; break }
	// - let TICKET = p.Ticket, COMMD = p.CommD
	ticket := []byte{}
	commD := []byte{}

	return s.builder.ReadPieceFromSealedSector(sectorId, 0, s.builder.SectorSize(), ticket, commD)
}

func (s *ChainBackedPieceManager) LocatePieceWithinSector(dealID uint64) (sectorID uint64, offset uint64, length uint64, err error) {
	/*

		// how the storage market actor structures sector id -> []DealID mapping

		type DealID int64
		type DealIDs struct {
			Items []DealID
		}

		type SectorPreCommitInfo struct {
		    SectorNumber  abi.SectorNumber
		    SealedCID     abi.SealedSectorCID  // CommR
		    SealEpoch     abi.ChainEpoch
		    DealIDs       abi.DealIDs
		    Expiration    abi.ChainEpoch
		}

		// where and how does the storage miner actor store SectorPreCommitInfo

		func (a *StorageMinerActorCode_I) PreCommitSector(rt Runtime, info SectorPreCommitInfo) {
			h, st := a.State(rt)
			rt.ValidateImmediateCallerIs(st.Info().Worker())

			if _, found := st.Sectors()[info.SectorNumber()]; found {
				rt.AbortStateMsg("Sector number already exists in table")
			}

			...

			newSectorInfo := &SectorOnChainInfo_I{
				State_:            SectorState_PreCommit,
				Info_:             info,
				PreCommitDeposit_: depositReq,
				PreCommitEpoch_:   rt.CurrEpoch(),
				ActivationEpoch_:  epochUndefined,
				DealWeight_:       *big.NewInt(-1),
			}
			st.Sectors()[info.SectorNumber()] = newSectorInfo

			...
		}

		// getting a deal from the storage market actor

		func (st *StorageMarketActorState_I) _getOnChainDeal(dealID abi.DealID) (
			deal OnChainDeal, dealP StorageDealProposal, ok bool) {

			var found bool
			deal, found = st.Deals()[dealID]
			if found {
				dealP = deal.Deal().Proposal()
			} else {
				deal = nil
				dealP = nil
			}
			return
		}

		type StorageDealProposal struct {
			PieceCID                      abi.PieceCID  // CommP
			PieceSize                     abi.PieceSize

			...
		}

		type StorageDeal struct {
			Proposal  StorageDealProposal
			CID()     &StorageDeal
		}

		type OnChainDeal struct {
			ID                abi.DealID
			Deal              StorageDeal
			...
		}



	*/

	// TODO:
	//
	// - let ID = `dealID`
	// - get slice of pre-committed sector info PSIS from storage miner actor storage
	// - let PSI = nil
	// - for each p in PSIS: for each d in p.DealIDs: if d.ID == ID then { PSI = p; break }
	// - for each d in PSI.DealIDs, call StorageMarketActor#_getOnChainDeal(d) to build a slice of OnChainDeal OS
	// - let DP = nil
	// - let sum = 0;
	// - for each o in OS: if o.ID == ID then { break } else { sum += o.Deal.Proposal.PieceSize }
	// - return (PSI.SectorNumber, sum, DP.PieceSize)

	return 0, 0, 0, nil
}

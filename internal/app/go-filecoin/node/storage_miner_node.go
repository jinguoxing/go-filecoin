package node

import (
	"context"
	"errors"
	"math"

	"github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-cid"
	"github.com/polydawn/refmt/cbor"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-filecoin/internal/pkg/block"
	"github.com/filecoin-project/go-filecoin/internal/pkg/types"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/abi"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
)

// TODO: lotus sets this value to the value of the Finality constant
const SealRandomnessLookback = 42

type StorageMinerNodeAPI interface {
	ChainHeadKey() block.TipSetKey
	ChainTipSet(key block.TipSetKey) (block.TipSet, error)
	ChainSampleRandomness(ctx context.Context, sampleHeight *types.BlockHeight) ([]byte, error)
	MessageSend(ctx context.Context, from, to address.Address, value types.AttoFIL, gasPrice types.AttoFIL, gasLimit types.GasUnits, method types.MethodID, params ...interface{}) (cid.Cid, chan error, error)
	MessageWait(ctx context.Context, msgCid cid.Cid, cb func(*block.Block, *types.SignedMessage, *types.MessageReceipt) error) error
	SignBytes(data []byte, addr address.Address) (types.Signature, error)
}

type StorageMinerNode struct {
	api    StorageMinerNodeAPI
	worker address.Address
	maddr  address.Address
}

var _ storage.NodeAPI = new(StorageMinerNode)

func NewStorageMinerNode(api StorageMinerNodeAPI, workerAddress address.Address, minerAddress address.Address) *StorageMinerNode {
	return &StorageMinerNode{
		api:    api,
		worker: workerAddress,
		maddr:  minerAddress,
	}
}
func (m *StorageMinerNode) SendSelfDeals(ctx context.Context, pieces ...storage.PieceInfo) (cid.Cid, error) {
	proposals := make([]types.StorageDealProposal, len(pieces))
	for i, piece := range pieces {
		proposals[i] = types.StorageDealProposal{
			PieceRef:             piece.CommP[:],
			PieceSize:            types.Uint64(piece.Size),
			Client:               m.worker,
			Provider:             m.maddr,
			ProposalExpiration:   math.MaxUint64,
			Duration:             math.MaxUint64 / 2, // /2 because overflows
			StoragePricePerEpoch: 0,
			StorageCollateral:    0,
			ProposerSignature:    nil,
		}

		proposalBytes, err := cbor.Marshal(proposals[i])
		if err != nil {
			return cid.Undef, err
		}

		sig, err := m.api.SignBytes(proposalBytes, m.worker)
		if err != nil {
			return cid.Undef, err
		}

		proposals[i].ProposerSignature = &sig
	}

	dealParams, err := abi.ToEncodedValues(proposals)
	if err != nil {
		return cid.Undef, err
	}

	mcid, cerr, err := m.api.MessageSend(
		ctx,
		m.worker,
		address.StorageMarketAddress,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		storagemarket.PublishStorageDeals,
		dealParams,
	)
	if err != nil {
		return cid.Undef, err
	}

	err = <-cerr
	if err != nil {
		return cid.Undef, err
	}

	return mcid, nil
}

func (m *StorageMinerNode) WaitForSelfDeals(ctx context.Context, mcid cid.Cid) ([]uint64, uint8, error) {
	receiptChan := make(chan *types.MessageReceipt)
	errChan := make(chan error)

	go func() {
		err := m.api.MessageWait(ctx, mcid, func(b *block.Block, message *types.SignedMessage, r *types.MessageReceipt) error {
			receiptChan <- r
			return nil
		})
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case receipt := <-receiptChan:
		if receipt.ExitCode != 0 {
			return nil, receipt.ExitCode, nil
		}

		dealIdValues, err := abi.Deserialize(receipt.Return[0], abi.UintArray)
		if err != nil {
			return nil, 0, err
		}

		dealIds, ok := dealIdValues.Val.([]uint64)
		if !ok {
			return nil, 0, errors.New("Decoded deal ids are not a []uint64")
		}

		return dealIds, 0, nil
	case err := <-errChan:
		return nil, 0, err
	case <-ctx.Done():
		return nil, 0, errors.New("context ended prematurely")
	}
}

func (m *StorageMinerNode) SendPreCommitSector(ctx context.Context, sectorID uint64, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error) {
	dealIds := make([]types.Uint64, len(pieces))
	for i, piece := range pieces {
		dealIds[i] = types.Uint64(piece.DealID)
	}

	info := &types.SectorPreCommitInfo{
		SectorNumber: types.Uint64(sectorID),

		CommR:     commR,
		SealEpoch: types.Uint64(ticket.BlockHeight),
		DealIDs:   dealIds,
	}

	precommitParams, err := abi.ToEncodedValues(info)
	if err != nil {
		return cid.Undef, err
	}

	mcid, cerr, err := m.api.MessageSend(
		ctx,
		m.worker,
		address.StorageMarketAddress,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		storagemarket.PreCommitSector,
		precommitParams,
	)
	if err != nil {
		return cid.Undef, err
	}

	err = <-cerr
	if err != nil {
		return cid.Undef, err
	}

	return mcid, nil
}

func (m *StorageMinerNode) WaitForPreCommitSector(ctx context.Context, mcid cid.Cid) (uint64, uint8, error) {
	return m.waitForMessageHeight(ctx, mcid)
}

func (m *StorageMinerNode) SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, deals ...uint64) (cid.Cid, error) {
	dealIds := make([]types.Uint64, len(deals))
	for i, deal := range deals {
		dealIds[i] = types.Uint64(deal)
	}

	info := &types.SectorProveCommitInfo{
		Proof:    proof,
		SectorID: types.Uint64(sectorID),
		DealIDs:  dealIds,
	}

	commitParams, err := abi.ToEncodedValues(info)
	if err != nil {
		return cid.Undef, err
	}

	mcid, cerr, err := m.api.MessageSend(
		ctx,
		m.worker,
		address.StorageMarketAddress,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		storagemarket.CommitSector,
		commitParams,
	)
	if err != nil {
		return cid.Undef, err
	}

	err = <-cerr
	if err != nil {
		return cid.Undef, err
	}

	return mcid, nil
}

func (m *StorageMinerNode) WaitForProveCommitSector(ctx context.Context, mcid cid.Cid) (uint64, uint8, error) {
	return m.waitForMessageHeight(ctx, mcid)
}

func (m *StorageMinerNode) GetSealTicket(ctx context.Context) (storage.SealTicket, error) {
	ts, err := m.api.ChainTipSet(m.api.ChainHeadKey())
	if err != nil {
		return storage.SealTicket{}, xerrors.Errorf("getting head ts for SealTicket failed: %w", err)
	}

	h, err := ts.Height()
	if err != nil {
		return storage.SealTicket{}, err
	}

	sampleAt := types.NewBlockHeight(h).Sub(types.NewBlockHeight(SealRandomnessLookback))

	r, err := m.api.ChainSampleRandomness(ctx, sampleAt)
	if err != nil {
		return storage.SealTicket{}, xerrors.Errorf("getting randomness for SealTicket failed: %w", err)
	}

	return storage.SealTicket{
		BlockHeight: h,
		TicketBytes: r,
	}, nil
}

func (m *StorageMinerNode) SetSealSeedHandler(ctx context.Context, preCommitMsg cid.Cid, seedAvailFunc func(storage.SealSeed), seedInvalidatedFunc func()) {
	panic("implement me")
}

type heightAndExitCode struct {
	exitCode uint8
	height   types.Uint64
}

func (m *StorageMinerNode) waitForMessageHeight(ctx context.Context, mcid cid.Cid) (uint64, uint8, error) {
	height := make(chan heightAndExitCode)
	errChan := make(chan error)

	go func() {
		err := m.api.MessageWait(ctx, mcid, func(b *block.Block, message *types.SignedMessage, r *types.MessageReceipt) error {
			height <- heightAndExitCode{
				height:   b.Height,
				exitCode: r.ExitCode,
			}
			return nil
		})
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case h := <-height:
		return uint64(h.height), h.exitCode, nil
	case err := <-errChan:
		return 0, 0, err
	case <-ctx.Done():
		return 0, 0, errors.New("context ended prematurely")
	}
}

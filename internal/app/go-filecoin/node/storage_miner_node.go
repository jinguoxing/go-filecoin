package node

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/abi"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/actor/builtin/storagemarket"
	"github.com/polydawn/refmt/cbor"

	"github.com/filecoin-project/go-filecoin/internal/pkg/block"
	"github.com/filecoin-project/go-filecoin/internal/pkg/types"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
	"github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-cid"
)

type StorageMinerNodeAPI interface {
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

func (m *StorageMinerNode) SendSelfDeals(ctx context.Context, pieces ...storage.PieceInfo) (cid.Cid, error) {
	proposals := make([]storagemarket.StorageDealProposal, len(pieces))
	for i, piece := range pieces {
		proposals[i] = storagemarket.StorageDealProposal{
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

func (m *StorageMinerNode) WaitForSelfDeals(ctx context.Context, mcid cid.Cid) ([]uint64, error) {
	var wg sync.WaitGroup
	var receipt *types.MessageReceipt

	wg.Add(1)
	go m.api.MessageWait(ctx, mcid, func(b *block.Block, message *types.SignedMessage, r *types.MessageReceipt) error {
		receipt = r
		wg.Done()
		return nil
	})

	wg.Wait()

	dealIdValues, err := abi.Deserialize(receipt.Return[0], abi.UintArray)
	if err != nil {
		return nil, err
	}

	dealIds, ok := dealIdValues.Val.([]uint64)
	if !ok {
		return nil, errors.New("Decoded deal ids are not a []uint64")
	}

	return dealIds, nil
}

func (StorageMinerNode) SendPreCommitSector(ctx context.Context, sectorID uint64, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error) {
	panic("implement me")
}

func (StorageMinerNode) WaitForPreCommitSector(context.Context, cid.Cid) (uint64, error) {
	panic("implement me")
}

func (StorageMinerNode) SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, dealids ...uint64) (cid.Cid, error) {
	panic("implement me")
}

func (StorageMinerNode) WaitForProveCommitSector(context.Context, cid.Cid) (uint64, error) {
	panic("implement me")
}

func (StorageMinerNode) GetSealTicket(context.Context) (storage.SealTicket, error) {
	panic("implement me")
}

func (StorageMinerNode) SetSealSeedHandler(ctx context.Context, preCommitMsg cid.Cid, seedAvailFunc func(storage.SealSeed), seedInvalidatedFunc func()) {
	panic("implement me")
}

func NewStorageMinernode(api StorageMinerNodeAPI) *StorageMinerNode {
	return &StorageMinerNode{
		api: api,
	}
}

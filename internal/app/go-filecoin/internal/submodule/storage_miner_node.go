package submodule

import (
	"context"
	"errors"
	"math"

	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/msg"

	"github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-cid"
	"github.com/polydawn/refmt/cbor"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-filecoin/internal/pkg/block"
	"github.com/filecoin-project/go-filecoin/internal/pkg/chain"
	"github.com/filecoin-project/go-filecoin/internal/pkg/types"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/abi"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
)

// TODO: lotus sets this value to the value of the Finality constant
const SealRandomnessLookbackEpochs = 42

type ChainSampler func(ctx context.Context, sampleHeight *types.BlockHeight) ([]byte, error)

type heightThresholdListener struct {
	target    uint64
	targetHit bool

	seedCh    chan storage.SealSeed
	errCh     chan error
	invalidCh chan struct{}
	doneCh    chan struct{}
}

// handle a chain update by sending appropriate status messages back to the channels.
// newChain is all the tipsets that are new since the last head update.
// Normally, this will be a single tipset, but in the case of a re-org it will contain
// All the common ancestors of the new tipset to the greatest common ancestor.
// Returns false if this handler is no longer valid
func (l heightThresholdListener) handle(ctx context.Context, chain []block.TipSet, sampler ChainSampler) (bool, error) {
	if len(chain) < 1 {
		return true, nil
	}

	h, err := chain[0].Height()
	if err != nil {
		return true, err
	}

	// check if we've hit finality and should stop listening
	if h >= l.target+SealRandomnessLookbackEpochs {
		return false, nil
	}

	lcaHeight, err := chain[len(chain)-1].Height()
	if err != nil {
		return true, err
	}

	// if we have already seen a target tipset
	if l.targetHit {
		// if we've completely reverted
		if h < l.target {
			l.invalidCh <- struct{}{}
			l.targetHit = false
			// if we've re-orged to a point before the target
		} else if lcaHeight < l.target {
			l.invalidCh <- struct{}{}
			l.sendRandomness(ctx, chain, sampler)
		}
		return true, nil
	}

	// otherwise send randomness if we've hit the height
	if h >= l.target {
		l.targetHit = true
		l.sendRandomness(ctx, chain, sampler)
	}
	return true, nil
}

func (l heightThresholdListener) sendRandomness(ctx context.Context, chain []block.TipSet, sampler ChainSampler) error {
	// assume chain not empty and first tipset height greater than target
	firstTargetTipset := chain[0]
	for _, ts := range chain {
		h, err := ts.Height()
		if err != nil {
			return err
		}

		if h < l.target {
			break
		}
		firstTargetTipset = ts
	}

	tsHeight, err := firstTargetTipset.Height()
	if err != nil {
		return err
	}

	randomness, err := sampler(ctx, types.NewBlockHeight(tsHeight))
	if err != nil {
		return err
	}

	l.seedCh <- storage.SealSeed{
		BlockHeight: tsHeight,
		TicketBytes: randomness,
	}
	return nil
}

type StorageMinerNode struct {
	minerAddr       address.Address
	workerAddr      address.Address
	newListener     chan heightThresholdListener
	heightListeners []heightThresholdListener

	chain     *ChainSubmodule
	messaging *MessagingSubmodule
	msgWaiter *msg.Waiter
	wallet    *WalletSubmodule
}

var _ storage.NodeAPI = new(StorageMinerNode)

func NewStorageMinerNode(minerAddress address.Address, workerAddress address.Address, c *ChainSubmodule, m *MessagingSubmodule, mw *msg.Waiter, w *WalletSubmodule) *StorageMinerNode {
	return &StorageMinerNode{
		minerAddr:  minerAddress,
		workerAddr: workerAddress,
		chain:      c,
		messaging:  m,
		msgWaiter:  mw,
		wallet:     w,
	}
}

func (m *StorageMinerNode) StartHeightListener(ctx context.Context, htc <-chan interface{}) {
	go func() {
		var previousHead block.TipSet
		for {
			select {
			case <-htc:
				head, err := m.handleNewTipSet(ctx, previousHead)
				if err != nil {
					log.Warn("failed to handle new tipset")
				} else {
					previousHead = head
				}
			case heightListener := <-m.newListener:
				m.heightListeners = append(m.heightListeners, heightListener)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *StorageMinerNode) handleNewTipSet(ctx context.Context, previousHead block.TipSet) (block.TipSet, error) {
	newHeadKey := m.chain.ChainReader.GetHead()
	newHead, err := m.chain.ChainReader.GetTipSet(newHeadKey)
	if err != nil {
		return block.TipSet{}, err
	}

	_, newTips, err := chain.CollectTipsToCommonAncestor(ctx, m.chain.ChainReader, previousHead, newHead)
	if err != nil {
		return block.TipSet{}, err
	}

	newListeners := make([]heightThresholdListener, len(m.heightListeners))
	for _, listener := range m.heightListeners {
		valid, err := listener.handle(ctx, newTips, m.chain.State.SampleRandomness)
		if err != nil {
			log.Error("Error checking storage miner chain listener", err)
		}

		if valid {
			newListeners = append(newListeners, listener)
		}
	}
	m.heightListeners = newListeners

	return newHead, nil
}

func (m *StorageMinerNode) SendSelfDeals(ctx context.Context, pieces ...storage.PieceInfo) (cid.Cid, error) {
	proposals := make([]types.StorageDealProposal, len(pieces))
	for i, piece := range pieces {
		proposals[i] = types.StorageDealProposal{
			PieceRef:             piece.CommP[:],
			PieceSize:            types.Uint64(piece.Size),
			Client:               m.workerAddr,
			Provider:             m.minerAddr,
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

		sig, err := m.wallet.Wallet.SignBytes(proposalBytes, m.workerAddr)
		if err != nil {
			return cid.Undef, err
		}

		proposals[i].ProposerSignature = &sig
	}

	dealParams, err := abi.ToEncodedValues(proposals)
	if err != nil {
		return cid.Undef, err
	}

	mcid, cerr, err := m.messaging.Outbox.Send(
		ctx,
		m.workerAddr,
		address.StorageMarketAddress,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		true,
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
		err := m.msgWaiter.Wait(ctx, mcid, func(b *block.Block, message *types.SignedMessage, r *types.MessageReceipt) error {
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

	mcid, cerr, err := m.messaging.Outbox.Send(
		ctx,
		m.workerAddr,
		m.minerAddr,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		true,
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

	mcid, cerr, err := m.messaging.Outbox.Send(
		ctx,
		m.workerAddr,
		m.minerAddr,
		types.ZeroAttoFIL,
		types.NewGasPrice(1),
		types.NewGasUnits(300),
		true,
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
	ts, err := m.chain.ChainReader.GetTipSet(m.chain.ChainReader.GetHead())
	if err != nil {
		return storage.SealTicket{}, xerrors.Errorf("getting head ts for SealTicket failed: %w", err)
	}

	h, err := ts.Height()
	if err != nil {
		return storage.SealTicket{}, err
	}

	r, err := m.chain.State.SampleRandomness(ctx, types.NewBlockHeight(h))
	if err != nil {
		return storage.SealTicket{}, xerrors.Errorf("getting randomness for SealTicket failed: %w", err)
	}

	return storage.SealTicket{
		BlockHeight: h,
		TicketBytes: r,
	}, nil
}

func (m *StorageMinerNode) GetSealSeed(ctx context.Context, preCommitMsg cid.Cid, interval uint64) (seed <-chan storage.SealSeed, err <-chan error, invalidated <-chan struct{}, done <-chan struct{}) {
	sc := make(chan storage.SealSeed)
	ec := make(chan error)
	ic := make(chan struct{})
	dc := make(chan struct{})
	go func() {
		h, exitCode, err := m.waitForMessageHeight(ctx, preCommitMsg)
		if err != nil {
			ec <- err
			return
		}

		if exitCode != 0 {
			ec <- xerrors.Errorf("non-zero exit code for pre-commit message %d", exitCode)
			return
		}

		m.newListener <- heightThresholdListener{
			target:    h + interval,
			targetHit: false,
			seedCh:    sc,
			errCh:     ec,
			invalidCh: ic,
			doneCh:    dc,
		}
	}()
	return sc, ec, ic, dc
}

type heightAndExitCode struct {
	exitCode uint8
	height   types.Uint64
}

func (m *StorageMinerNode) waitForMessageHeight(ctx context.Context, mcid cid.Cid) (uint64, uint8, error) {
	height := make(chan heightAndExitCode)
	errChan := make(chan error)

	go func() {
		err := m.msgWaiter.Wait(ctx, mcid, func(b *block.Block, message *types.SignedMessage, r *types.MessageReceipt) error {
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

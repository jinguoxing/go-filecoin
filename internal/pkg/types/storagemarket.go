package types

import "github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"

type StorageDealProposal struct {
	PieceRef  []byte // cid bytes // TODO: spec says to use cid.Cid, probably not a good idea
	PieceSize Uint64

	Client   address.Address
	Provider address.Address

	ProposalExpiration Uint64
	Duration           Uint64

	StoragePricePerEpoch Uint64
	StorageCollateral    Uint64

	ProposerSignature *Signature
}

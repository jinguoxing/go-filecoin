package verification

import (
	"context"
	"github.com/filecoin-project/go-filecoin/internal/pkg/types"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
	go_sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
)

// VerifySealRequest represents a request to verify the output of a Seal() operation.
type VerifySealRequest struct {
	CommD         types.CommD      // returned from seal
	CommR         types.CommR      // returned from seal
	CommRStar     types.CommRStar  // returned from seal
	Proof         types.PoRepProof // returned from seal
	ProverAddress address.Address  // uniquely identifies miner
	Ticket        []byte           // ticket for VRF
	Seed          []byte           // seed for VRF
	SectorID      uint64           // uniquely identifies sector
	SectorSize    *types.BytesAmount
}

// VerifyPoStRequest represents a request to generate verify a proof-of-spacetime.
type VerifyPoStRequest struct {
	ChallengeSeed    types.PoStChallengeSeed
	SortedSectorInfo go_sectorbuilder.SortedPublicSectorInfo
	Candidates       []go_sectorbuilder.EPostCandidate
	ProverAddress    address.Address
	Faults           []uint64
	Proof            types.PoStProof
	SectorSize       *types.BytesAmount
}

// VerifyPoStResponse communicates the validity of a provided proof-of-spacetime.
type VerifyPoStResponse struct {
	IsValid bool
}

// VerifySealResponse communicates the validity of a provided proof-of-replication.
type VerifySealResponse struct {
	IsValid bool
}

// VerifyPieceInclusionProofRequest represents a request to verify a piece
// inclusion proof.
type VerifyPieceInclusionProofRequest struct {
	CommD               types.CommD
	CommP               types.CommP
	PieceInclusionProof []byte
	PieceSize           *types.BytesAmount
	SectorSize          *types.BytesAmount
}

// VerifyPieceInclusionProofResponse communicates the validity of a provided
// piece inclusion proof.
type VerifyPieceInclusionProofResponse struct {
	IsValid bool
}

// Verifier provides an interface to the proving subsystem.
type Verifier interface {
	VerifyFallbackPoSt(context.Context, VerifyPoStRequest) (VerifyPoStResponse, error)
	VerifySeal(VerifySealRequest) (VerifySealResponse, error)
}

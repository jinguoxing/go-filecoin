package verification

import (
	"context"
	"github.com/filecoin-project/go-address"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
)

// RustVerifier provides proof-verification methods.
type RustVerifier struct{}

var _ Verifier = &RustVerifier{}

// VerifySeal returns nil if the Seal operation from which its inputs were
// derived was valid, and an error if not.
func (rp *RustVerifier) VerifySeal(req VerifySealRequest) (VerifySealResponse, error) {
	prover, err := address.NewFromBytes(req.ProverAddress.Bytes())
	if err != nil {
		return VerifySealResponse{}, err
	}
	isValid, err := sectorbuilder.ProofVerifier.VerifySeal(req.SectorSize.Uint64(), req.CommR[:], req.CommD[:], prover, req.Ticket, req.Seed, req.SectorID, req.Proof)
	if err != nil {
		return VerifySealResponse{}, err
	}

	return VerifySealResponse{
		IsValid: isValid,
	}, nil
}

// VerifyFallbackPoSt verifies that a proof-of-spacetime is valid.
func (rp *RustVerifier) VerifyFallbackPoSt(ctx context.Context, req VerifyPoStRequest) (VerifyPoStResponse, error) {
	sbAddress, err := address.NewFromBytes(req.ProverAddress.Bytes())
	if err != nil {
		return VerifyPoStResponse{}, err
	}

	isValid, err := sectorbuilder.ProofVerifier.VerifyFallbackPost(ctx, req.SectorSize.Uint64(), req.SortedSectorInfo, req.ChallengeSeed[:], req.Proof, req.Candidates, sbAddress, uint64(len(req.Faults)))
	if err != nil {
		return VerifyPoStResponse{}, err
	}

	return VerifyPoStResponse{
		IsValid: isValid,
	}, nil
}

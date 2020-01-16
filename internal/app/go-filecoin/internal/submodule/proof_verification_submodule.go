package submodule

import (
	"github.com/filecoin-project/go-filecoin/internal/pkg/proofs/verification"
)

type ProofVerificationSubmodule struct {
	ProofVerifier verification.Verifier
}

// NewProofVerificationSubmodule creates a new proof verification submodule
func NewProofVerificationSubmodule() ProofVerificationSubmodule {
	return ProofVerificationSubmodule{
		ProofVerifier: verification.NewFFIBackedProofVerifier(),
	}
}

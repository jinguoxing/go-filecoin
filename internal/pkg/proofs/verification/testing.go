package verification

import "context"

// FakeVerifier is a simple mock Verifier for testing.
type FakeVerifier struct {
	VerifyPoStValid                bool
	VerifyPoStError                error
	VerifyPieceInclusionProofValid bool
	VerifyPieceInclusionProofError error
	VerifySealValid                bool
	VerifySealError                error

	// these requests will be captured by code that calls VerifySeal or VerifyFallbackPoSt or VerifyPieceInclusionProof
	LastReceivedVerifySealRequest                *VerifySealRequest
	LastReceivedVerifyPoStRequest                *VerifyPoStRequest
	LastReceivedVerifyPieceInclusionProofRequest *VerifyPieceInclusionProofRequest
}

var _ Verifier = (*FakeVerifier)(nil)

// VerifyFallbackPoSt fakes out PoSt proof verification, skipping an FFI call to
// generate_post.
func (fp *FakeVerifier) VerifyFallbackPoSt(ctx context.Context, req VerifyPoStRequest) (VerifyPoStResponse, error) {
	fp.LastReceivedVerifyPoStRequest = &req
	return VerifyPoStResponse{IsValid: fp.VerifyPoStValid}, fp.VerifyPoStError
}

// VerifySeal fakes out seal (PoRep) proof verification, skipping an FFI call to
// verify_seal.
func (fp *FakeVerifier) VerifySeal(req VerifySealRequest) (VerifySealResponse, error) {
	fp.LastReceivedVerifySealRequest = &req
	return VerifySealResponse{IsValid: fp.VerifySealValid}, fp.VerifySealError
}

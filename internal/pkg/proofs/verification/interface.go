package verification

import (
	ffi "github.com/filecoin-project/filecoin-ffi"
)

// Verifier provides an interface to the proving subsystem.
type Verifier interface {
	VerifySeal(
		sectorSize uint64,
		commR [ffi.CommitmentBytesLen]byte,
		commD [ffi.CommitmentBytesLen]byte,
		proverID [32]byte,
		ticket [32]byte,
		seed [32]byte,
		sectorID uint64,
		proof []byte,
	) (bool, error)

	VerifyPoSt(
		sectorSize uint64,
		sectorInfo ffi.SortedPublicSectorInfo,
		randomness [32]byte,
		challengeCount uint64,
		proof []byte,
		winners []ffi.Candidate,
		proverID [32]byte,
	) (bool, error)
}

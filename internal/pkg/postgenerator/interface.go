package postgenerator

import ffi "github.com/filecoin-project/filecoin-ffi"

type PoStGenerator interface {
	GenerateEPostCandidates(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed [ffi.CommitmentBytesLen]byte, faults []uint64) ([]ffi.Candidate, error)
	GenerateFallbackPoSt(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed [ffi.CommitmentBytesLen]byte, faults []uint64) ([]ffi.Candidate, []byte, error)
	ComputeElectionPoSt(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed []byte, winners []ffi.Candidate) ([]byte, error)
}

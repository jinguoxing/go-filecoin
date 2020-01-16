package postgenerator

import (
	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-sectorbuilder"
)

type SectorBuilderBackedPoStGenerator struct {
	builder sectorbuilder.Interface
}

func (s *SectorBuilderBackedPoStGenerator) GenerateEPostCandidates(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed [ffi.CommitmentBytesLen]byte, faults []uint64) ([]ffi.Candidate, error) {
	return s.builder.GenerateEPostCandidates(sectorInfo, challengeSeed, faults)
}

func (s *SectorBuilderBackedPoStGenerator) GenerateFallbackPoSt(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed [ffi.CommitmentBytesLen]byte, faults []uint64) ([]ffi.Candidate, []byte, error) {
	return s.builder.GenerateFallbackPoSt(sectorInfo, challengeSeed, faults)
}

func (s *SectorBuilderBackedPoStGenerator) ComputeElectionPoSt(sectorInfo ffi.SortedPublicSectorInfo, challengeSeed []byte, winners []ffi.Candidate) ([]byte, error) {
	return s.builder.ComputeElectionPoSt(sectorInfo, challengeSeed, winners)
}

func NewSectorBuilderBackedPoStGenerator(s sectorbuilder.Interface) *SectorBuilderBackedPoStGenerator {
	return &SectorBuilderBackedPoStGenerator{builder: s}
}

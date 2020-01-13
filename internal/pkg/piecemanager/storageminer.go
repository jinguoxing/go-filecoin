package piecemanager

import (
	"context"
	"github.com/filecoin-project/go-filecoin/internal/pkg/vm/address"
	sm "github.com/filecoin-project/go-storage-miner"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"io"
)

var log = logging.Logger("rustsectorbuilder") // nolint: deadcode

// MaxNumStagedSectors configures the maximum number of staged sectors which can
// be open and accepting data at any time.
const MaxNumStagedSectors = 1

// StorageMiner is a struct which serves as a proxy for a PieceManager in Rust.
type StorageMiner struct {
	miner *sm.Miner
}

var _ PieceManager = &StorageMiner{}

// StorageMinerConfig is a configuration object used when instantiating a
// Rust-backed PieceManager through the FFI. All fields are required.
type StorageMinerConfig struct {
	LastUsedSectorID uint64
	MetadataDir      string
	MinerAddr        address.Address
	SealedSectorDir  string
	StagedSectorDir  string
}

// NewStorageMiner instantiates a PieceManager through the FFI.
func NewStorageMiner(api sm.NodeAPI, ds datastore.Batching, sb sm.SectorBuilderAPI) (*StorageMiner, error) {
	m, err := sm.NewMiner(api, ds, sb)
	if err != nil {
		return nil, err
	}
	return &StorageMiner{
		miner: m,
	}, nil
}

// AddPiece writes the given piece into an unsealed sector and returns the id
// of that sector.
func (sb *StorageMiner) AddPiece(ctx context.Context, dealID uint64, pieceSize uint64, pieceReader io.Reader) (sectorID uint64, retErr error) {
	sectorID, _, err := sb.miner.AllocatePiece(pieceSize)
	if err != nil {
		return 0, err
	}

	return sectorID, sb.miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, dealID)
}

// ReadPieceFromSealedSector produces a Reader used to get original piece-bytes
// from a sealed sector.
func (sb *StorageMiner) ReadPieceFromSealedSector(pieceCid cid.Cid) (io.Reader, error) {
	panic("we can't do this right now")
	//buffer, err := go_sectorbuilder.ReadPieceFromSealedSector(sb.ptr, pieceCid.String())
	//if err != nil {
	//	return nil, err
	//}
	//
	//return bytes.NewReader(buffer), err
}

// Close closes the sector builder and deallocates its (Rust) memory, rendering
// it unusable for I/O.
func (sb *StorageMiner) Close() error {
	return nil
}

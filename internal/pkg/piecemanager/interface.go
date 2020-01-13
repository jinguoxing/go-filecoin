package piecemanager

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// PieceManager provides an interface through which user piece-bytes can be
// written, sealed into sectors, and later unsealed and read.
type PieceManager interface {
	// AddPiece writes the given piece into an unsealed sector and returns the
	// id of that sector. This method has a race; it is possible that the
	// sector into which the piece-bytes were written is sealed before this
	// method returns. In the real world this should not happen, as sealing
	// takes a long time to complete. In tests, where sealing happens
	// near-instantaneously, it is possible to exercise this race.
	AddPiece(ctx context.Context, dealID uint64, pieceSize uint64, pieceReader io.Reader) (sectorID uint64, err error)

	// ReadPieceFromSealedSector produces a Reader used to get original
	// piece-bytes from a sealed sector.
	ReadPieceFromSealedSector(pieceCid cid.Cid) (io.Reader, error)

	// Close signals that this PieceManager is no longer in use. PieceManager
	// metadata will not be deleted when Close is called; an equivalent
	// PieceManager can be created later by applying the Init function to the
	// arguments used to create the instance being closed.
	Close() error
}

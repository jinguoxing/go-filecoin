package submodule

import (
	"context"

	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
)

// PieceManagerSubmodule enhances the `Node` with sector storage capabilities.
type PieceManagerSubmodule struct {
	// PieceManager is used by the miner to fill and seal sectors.
	PieceManager piecemanager.PieceManager
}

// NewPieceManagerSubmodule creates a new sector builder submodule.
func NewPieceManagerSubmodule(ctx context.Context) (PieceManagerSubmodule, error) {
	return PieceManagerSubmodule{
		// sectorBuilder: nil,
	}, nil
}

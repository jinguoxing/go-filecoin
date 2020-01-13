package submodule

import (
	"context"

	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
)

// SectorBuilderSubmodule enhances the `Node` with sector storage capabilities.
type SectorBuilderSubmodule struct {
	// PieceManager is used by the miner to fill and seal sectors.
	SectorBuilder piecemanager.PieceManager
}

// NewSectorStorageSubmodule creates a new sector builder submodule.
func NewSectorStorageSubmodule(ctx context.Context) (SectorBuilderSubmodule, error) {
	return SectorBuilderSubmodule{
		// sectorBuilder: nil,
	}, nil
}

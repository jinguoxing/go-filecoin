package porcelain

import (
	"context"
	"io"

	"github.com/filecoin-project/go-filecoin/internal/pkg/piecemanager"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"

	"github.com/pkg/errors"
)

type sbPlumbing interface {
	SectorBuilder() piecemanager.PieceManager
	DAGImportData(context.Context, io.Reader) (ipld.Node, error)
	DAGCat(context.Context, cid.Cid) (io.Reader, error)
	DAGGetFileSize(ctx context.Context, c cid.Cid) (uint64, error)
}

// AddPiece adds piece data to a staged sector
func AddPiece(ctx context.Context, plumbing sbPlumbing, dealID uint64, pieceReader io.Reader) (uint64, error) {
	if plumbing.SectorBuilder() == nil {
		return 0, errors.New("must be mining to add piece")
	}

	node, err := plumbing.DAGImportData(ctx, pieceReader)
	if err != nil {
		return 0, errors.Wrap(err, "could not read piece into local store")
	}

	dagReader, err := plumbing.DAGCat(ctx, node.Cid())
	if err != nil {
		return 0, errors.Wrap(err, "could not obtain reader for piece")
	}

	size, err := plumbing.DAGGetFileSize(ctx, node.Cid())
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate piece size")
	}

	sectorID, err := plumbing.SectorBuilder().AddPiece(ctx, dealID, size, dagReader)
	if err != nil {
		return 0, errors.Wrap(err, "could not add piece")
	}

	return sectorID, nil
}

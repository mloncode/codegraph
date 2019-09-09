package codegraph

import (
	"context"
	"io"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/quad/nquads"
)

const (
	defaultKV = "bolt"
)

// Graph is a opaque type for cayley graph database handler.
type Graph struct {
	store *cayley.Handle
}

// Open opens git graph database
func Open(dbpath string) (*Graph, error) {
	err := graph.InitQuadStore(defaultKV, dbpath, nil)
	if err != nil && err != graph.ErrDatabaseExists {
		return nil, err
	}

	store, err := cayley.NewGraph(defaultKV, dbpath, nil)
	if err != nil {
		return nil, err
	}

	return &Graph{store}, nil
}

// Import imports git repository into graph database
// Returns number of imported commits
func (g *Graph) Import(_ context.Context, gitpath string, opts *ExportOptions) error {
	w := graph.NewWriter(g.store)
	defer w.Close()

	exp, err := NewExporter(w, opts)
	if err != nil {
		return err
	}
	err = exp.ExportRepoPath(gitpath)
	if cerr := exp.Close(); err == nil {
		err = cerr
	}
	return err
}

// Export exports quads in raw format
// Returns number of exported quads
func (g *Graph) Export(ctx context.Context, w io.Writer) (int, error) {
	qw := nquads.NewWriter(w)
	defer qw.Close()

	it, _ := g.store.QuadsAllIterator().Optimize()
	defer it.Close()

	n := 0
	for it.Next(ctx) {
		q := g.store.Quad(it.Result())
		if err := qw.WriteQuad(q); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}

// Close implements io.Closer
func (g *Graph) Close() error {
	if g != nil && g.store != nil {
		return g.store.Close()
	}
	return nil
}

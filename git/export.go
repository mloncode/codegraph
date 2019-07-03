package git

import (
	"context"
	"io"

	"github.com/cayleygraph/cayley/quad/nquads"
)

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

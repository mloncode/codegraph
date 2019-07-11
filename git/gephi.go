package git

import "github.com/cayleygraph/cayley/quad"

const (
	predGephiInline = quad.IRI("gephi:inline")
)

func writeGephiMetadata(w quad.Writer) error {
	for _, pred := range []quad.IRI{
		prdMetadata,
		prdMessage,
	} {
		if err := w.WriteQuad(quad.Quad{
			Subject:   pred,
			Predicate: predGephiInline,
			Object:    quad.Bool(true),
		}); err != nil {
			return err
		}
	}
	return nil
}

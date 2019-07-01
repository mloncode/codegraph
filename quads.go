package codegraph

import (
	"fmt"
	"strings"

	"github.com/bblfsh/sdk/v3/uast"
	"github.com/bblfsh/sdk/v3/uast/nodes"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/voc/rdf"
)

const (
	predUAST = quad.IRI("uast:Root")
	predRole = quad.IRI("uast:Role")
	predPos  = quad.IRI("uast:Pos")
	predFile = quad.IRI("uast:File")
)

func AsQuads(w quad.Writer, file quad.Value, n nodes.Node) error {
	ids, err := writeNodeQuads(w, file, n)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := w.WriteQuad(quad.Quad{
			Subject:   file,
			Predicate: predUAST,
			Object:    id,
		}); err != nil {
			return err
		}
	}
	return nil
}

var seenBNodes = make(map[quad.BNode]struct{})

func nextID() quad.Value {
	for {
		id := quad.RandomBlankNode()
		if _, ok := seenBNodes[id]; !ok {
			seenBNodes[id] = struct{}{}
			return id
		}
	}
}

func writeNodeQuads(w quad.Writer, file quad.Value, n nodes.Node) ([]quad.Value, error) {
	switch n := n.(type) {
	case nil:
		return nil, nil
	case nodes.String:
		if n == "" {
			return nil, nil
		}
		return []quad.Value{quad.String(n)}, nil
	case nodes.Int:
		return []quad.Value{quad.Int(n)}, nil
	case nodes.Uint:
		return []quad.Value{quad.Int(n)}, nil // TODO: uint
	case nodes.Float:
		return []quad.Value{quad.Float(n)}, nil
	case nodes.Bool:
		return []quad.Value{quad.Bool(n)}, nil
	case nodes.Array:
		out := make([]quad.Value, 0, len(n))
		for _, v := range n {
			vid, err := writeNodeQuads(w, file, v)
			if err != nil {
				return nil, err
			}
			out = append(out, vid...)
		}
		return out, nil
	case nodes.Object:
		id := nextID()
		typ := uast.TypeOf(n)
		if typ == uast.TypePosition {
			// add a file reference to positions
			if err := w.WriteQuad(quad.Quad{
				Subject:   id,
				Predicate: predFile,
				Object:    file,
			}); err != nil {
				return nil, err
			}
		}
		ns := ""
		if i := strings.Index(typ, ":"); i > 0 {
			ns = typ[:i+1]
		}
		for k, v := range n {
			sub, err := writeNodeQuads(w, file, v)
			if err != nil {
				return nil, err
			}
			pred := quad.IRI(k)
			switch k {
			case uast.KeyType:
				pred = rdf.Type
				if len(sub) == 1 {
					if t, ok := sub[0].(quad.String); ok {
						sub[0] = quad.IRI(t)
					}
				}
			case uast.KeyRoles:
				pred = predRole
			case uast.KeyPos:
				pred = predPos
			default:
				if !strings.Contains(k, ":") && ns != "" {
					pred = quad.IRI(ns + k)
				}
			}
			for _, vid := range sub {
				if err = w.WriteQuad(quad.Quad{
					Subject:   id,
					Predicate: pred,
					Object:    vid,
				}); err != nil {
					return nil, err
				}
			}
		}
		return []quad.Value{id}, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", n)
	}
}

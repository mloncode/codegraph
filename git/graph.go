package git

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/voc/rdf"
	"github.com/cayleygraph/cayley/voc/schema"
)

const (
	defaultKV = "bolt"

	// typeRepo node keep (back) references to all to level nodes (so far used only for repos)
	typeRepo   = quad.IRI("git:Repo")
	typeCommit = quad.IRI("git:Commit")
	typeFile   = quad.IRI("git:File")
	typeAuthor = quad.IRI("git:Author")

	// node type predicate
	prdType = quad.IRI(rdf.Type)

	// commits
	prdCommit   = quad.IRI("git:commit")
	prdMetadata = quad.IRI("git:metadata")
	prdMessage  = quad.IRI("git:message")
	prdAuthor   = quad.IRI("git:author")
	prdCommiter = quad.IRI("git:commiter")
	prdName     = quad.IRI(schema.Name)
	prdEmail    = quad.IRI("git:email")
	prdChild    = quad.IRI("git:child")
	prdParent   = quad.IRI("git:parent")

	// files
	prdFile   = quad.IRI("git:file")
	prdAdd    = quad.IRI("git:add")
	prdRemove = quad.IRI("git:remove")
	prdModify = quad.IRI("git:modify")
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

// Close closes git graph database (GitGraph is io.Closer)
func Close(g *Graph) error {
	return g.Close()
}

// Close implements io.Closer
func (g *Graph) Close() error {
	if g.store != nil {
		return g.store.Close()
	}
	return nil
}
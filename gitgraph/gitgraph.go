package gitgraph

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/quad"
)

const (
	defaultKV = "bolt"

	// nodeType node keep references to all to level nodes (so far used only for repos)
	nodeType = quad.IRI("type")

	// repo predicate
	prdRepo = quad.IRI("repo")

	// commits
	prdCommit = quad.IRI("commit")
	prdChild  = quad.IRI("child")
	prdParent = quad.IRI("parent")

	// files
	prdFile   = quad.IRI("file")
	prdAdd    = quad.IRI("add")
	prdRemove = quad.IRI("remove")
	prdModify = quad.IRI("modify")
)

// GitGraph is a opaque type for graph database handler.
type GitGraph struct {
	store *cayley.Handle
}

// Open opens git graph database
func Open(dbpath string) (*GitGraph, error) {
	err := graph.InitQuadStore(defaultKV, dbpath, nil)
	if err != nil && err != graph.ErrDatabaseExists {
		return nil, err
	}

	store, err := cayley.NewGraph(defaultKV, dbpath, nil)
	if err != nil {
		return nil, err
	}

	return &GitGraph{store}, nil
}

// Close closes git graph database (GitGraph is io.Closer)
func Close(g *GitGraph) error {
	return g.Close()
}

// Close implements io.Closer
func (g *GitGraph) Close() error {
	if g.store != nil {
		return g.store.Close()
	}
	return nil
}

package git

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/voc/rdf"
)

const (
	defaultKV = "bolt"

	// typeRepo node keep (back) references to all to level nodes (so far used only for repos)
	typeRepo = quad.IRI("git:repo")

	// node type predicate
	prdType = quad.IRI(rdf.Type)

	// commits
	prdCommit        = quad.IRI("git:commit")
	prdMetadata      = quad.IRI("git:metadata")
	prdMessage       = quad.IRI("git:message")
	prdAuthorName    = quad.IRI("git:authorName")
	prdAuthorEmail   = quad.IRI("git:authorEmail")
	prdAuthorTS      = quad.IRI("git:authorTS")
	prdCommiterName  = quad.IRI("git:commiterName")
	prdCommiterEmail = quad.IRI("git:commiterEmail")
	prdCommiterTS    = quad.IRI("git:commiterTS")
	prdChild         = quad.IRI("git:child")
	prdParent        = quad.IRI("git:parent")

	// files
	prdFile   = quad.IRI("git:file")
	prdAdd    = quad.IRI("git:add")
	prdRemove = quad.IRI("git:remove")
	prdModify = quad.IRI("git:modify")
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

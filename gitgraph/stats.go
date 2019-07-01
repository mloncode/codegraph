package gitgraph

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/path"
	"github.com/cayleygraph/cayley/graph/shape"
	"github.com/cayleygraph/cayley/quad"
)

type (
	// CommitStats contains commit statistics
	CommitStats struct {
		Hash       string // commit hash
		Label      string // metadata
		NumParents int    // number of parents

		NumFiles    int // number of files
		NumAdded    int // number of added files by this commit
		NumRemoved  int // number of removed files by this commit
		NumModified int // number of modified files by this commit
	}

	// SortBy is a function to sort commit statistics
	SortBy func(cs1, cs2 *CommitStats) bool

	commitStatsSorter struct {
		stats []*CommitStats
		by    SortBy
	}
)

// PrintStats prints commit statistics
func (g *GitGraph) PrintStats(ctx context.Context, limit int, by SortBy, nomerge bool) error {
	it, _ := cayley.StartPath(g.store, nodeType).Out(prdRepo).BuildIterator().Optimize()
	it, _ = g.store.OptimizeIterator(it)
	defer it.Close()

	for it.Next(ctx) {
		repo := g.store.NameOf(it.Result())
		if err := printStats(ctx, g.store, repo, limit, by, nomerge); err != nil {
			return err
		}
	}

	return nil
}

func printStats(ctx context.Context, qs graph.QuadStore, repo quad.Value, limit int, by SortBy, nomerge bool) error {
	var stats []*CommitStats

	it, _ := cayley.StartPath(qs, repo).Out(prdCommit).BuildIterator().Optimize()
	it, _ = qs.OptimizeIterator(it)
	for it.Next(ctx) {
		commit := qs.NameOf(it.Result())
		stats = append(stats, commitStats(ctx, qs, commit))
	}

	fmt.Printf("\n%s\n", repo.String())
	by.Sort(stats)
	n := 0
	for _, s := range stats {
		if limit > 0 && n >= limit {
			break
		}
		if s.NumParents > 1 && nomerge {
			continue
		}

		fmt.Printf("--\ncommit: %s", s.Hash)
		if s.NumParents > 1 {
			fmt.Print(" (merge)")
		}
		fmt.Printf("\n%s\n", strings.ReplaceAll(s.Label, `\n`, "\n"))

		touch := s.NumAdded + s.NumRemoved + s.NumModified
		fmt.Printf("%d files, %d touched (+, -, #), %d added(+), %d removed(-), %d modified(#)\n", s.NumFiles, touch, s.NumAdded, s.NumRemoved, s.NumModified)
		n++
	}

	return it.Close()
}

func commitStats(ctx context.Context, qs graph.QuadStore, commit quad.Value) *CommitStats {
	cs := &CommitStats{
		Hash: commit.String(),
	}

	sh := shape.Quads{
		{Dir: quad.Predicate, Values: shape.Lookup{prdCommit}},
		{Dir: quad.Object, Values: shape.Lookup{commit}},
	}

	it, _ := sh.BuildIterator(qs).Optimize()
	it, _ = qs.OptimizeIterator(it)
	if it.Next(ctx) {
		if lbl := qs.Quad(it.Result()).Label; lbl != nil {
			cs.Label = lbl.String()
		}
	}
	it.Close()

	path := cayley.StartPath(qs, commit)
	cs.NumParents = countPaths(ctx, qs, path.Out(prdParent))
	cs.NumFiles = countPaths(ctx, qs, path.Out(prdFile))
	cs.NumAdded = countPaths(ctx, qs, path.In(prdAdd))
	cs.NumRemoved = countPaths(ctx, qs, path.In(prdRemove))
	cs.NumModified = countPaths(ctx, qs, path.In(prdModify))
	return cs
}

func countPaths(ctx context.Context, qs graph.QuadStore, path *path.Path) int {
	n := 0

	it, _ := path.BuildIterator().Optimize()
	it, _ = qs.OptimizeIterator(it)
	for it.Next(ctx) {
		n++
		for it.NextPath(ctx) {
			n++
		}
	}
	it.Close()

	return n
}

// Len is part of sort.Interface.
func (cs *commitStatsSorter) Len() int {
	return len(cs.stats)
}

// Swap is part of sort.Interface.
func (cs *commitStatsSorter) Swap(i, j int) {
	cs.stats[i], cs.stats[j] = cs.stats[j], cs.stats[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (cs *commitStatsSorter) Less(i, j int) bool {
	return cs.by(cs.stats[i], cs.stats[j])
}

//Sort sorts
func (by SortBy) Sort(stats []*CommitStats) {
	sort.Sort(&commitStatsSorter{
		stats: stats,
		by:    by,
	})
}

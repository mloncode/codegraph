package gitgraph

import (
	"context"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/quad"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

// Import imports git repository into graph database
// Returns number of imported commits
func (g *GitGraph) Import(ctx context.Context, gitpath string) (int, error) {
	repo, repoIRI, err := openGit(gitpath)
	if err != nil {
		return 0, err
	}

	w := graph.NewWriter(g.store)
	defer w.Close()

	if err = w.WriteQuad(quad.Quad{
		Subject:   nodeType,
		Predicate: prdRepo,
		Object:    repoIRI,
	}); err != nil {
		return 0, err
	}

	// import commits
	it, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return 0, err
	}
	defer it.Close()

	n := 0
	err = it.ForEach(func(c *object.Commit) error {
		n++
		return importCommit(w, repoIRI, c)
	})
	return n, err
}

func openGit(gitpath string) (*git.Repository, quad.IRI, error) {
	repoIRI := quad.IRI(gitpath)

	// We instanciate a new repository targeting the given path (the .git folder)
	repo, err := git.PlainOpen(gitpath)
	if err != nil {
		return nil, "", err
	}
	if origin, err := repo.Remote("origin"); err == nil {
		if origin != nil && len(origin.Config().URLs) > 0 {
			repoIRI = quad.IRI(origin.Config().URLs[0])
		}
	}

	return repo, repoIRI, nil
}

func importCommit(w graph.BatchWriter, repoIRI quad.IRI, commit *object.Commit) error {
	commitIRI := quad.IRI(commit.Hash.String())

	if err := w.WriteQuad(quad.Quad{
		Subject:   repoIRI,
		Predicate: prdCommit,
		Object:    commitIRI,
		Label:     quad.String(commit.String()),
	}); err != nil {
		return err
	}

	// dump parents
	for _, p := range commit.ParentHashes {
		if _, err := w.WriteQuads([]quad.Quad{
			quad.Quad{
				Subject:   commitIRI,
				Predicate: prdParent,
				Object:    quad.IRI(p.String()),
			},
			quad.Quad{
				Subject:   quad.IRI(p.String()),
				Predicate: prdChild,
				Object:    commitIRI,
			},
		}); err != nil {
			return err
		}
	}

	// import files
	it, err := commit.Files()
	if err != nil {
		return err
	}
	defer it.Close()
	if err := it.ForEach(func(f *object.File) error {
		return importFile(w, commitIRI, f)
	}); err != nil {
		return err
	}

	// dump changes
	var (
		from *object.Tree
		to   *object.Tree
	)
	if commit.NumParents() > 0 {
		p, err := commit.Parent(0)
		if err != nil {
			return err
		}
		if from, err = p.Tree(); err != nil {
			return err
		}
	}
	if to, err = commit.Tree(); err != nil {
		return err
	}
	changes, err := object.DiffTree(from, to)
	if err != nil {
		return err
	}
	for _, ch := range changes {
		if err = importChange(w, commitIRI, ch); err != nil {
			return err
		}
	}

	return nil
}

func importFile(w graph.BatchWriter, commitIRI quad.IRI, file *object.File) error {
	fileIRI := quad.IRI(file.Hash.String())

	return w.WriteQuad(quad.Quad{
		Subject:   commitIRI,
		Predicate: prdFile,
		Object:    fileIRI,
		Label:     quad.String(file.Name),
	})
}

func importChange(w graph.BatchWriter, commitIRI quad.IRI, change *object.Change) error {
	action, err := change.Action()
	if err != nil {
		return err
	}

	fromIRI := quad.IRI(change.From.TreeEntry.Hash.String())
	toIRI := quad.IRI(change.To.TreeEntry.Hash.String())

	switch action {
	case merkletrie.Delete:
		err = w.WriteQuad(quad.Quad{
			Subject:   fromIRI,
			Predicate: prdRemove,
			Object:    commitIRI,
			Label:     quad.String(change.From.Name),
		})
	case merkletrie.Insert:
		err = w.WriteQuad(quad.Quad{
			Subject:   toIRI,
			Predicate: prdAdd,
			Object:    commitIRI,
			Label:     quad.String(change.To.Name),
		})
	case merkletrie.Modify:
		err = w.WriteQuad(quad.Quad{
			Subject:   toIRI,
			Predicate: prdModify,
			Object:    commitIRI,
			Label:     quad.String(change.To.Name),
		})
	}

	return err
}

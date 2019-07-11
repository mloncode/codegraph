package git

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/quad"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

// AsQuads writes a git repository history as a set of quads.
// It returns the number of commits processed.
func AsQuads(w quad.Writer, gitpath string) (int, error) {
	repo, repoIRI, err := openGit(gitpath)
	if err != nil {
		return 0, err
	}
	if err = writeGephiMetadata(w); err != nil {
		return 0, err
	}

	if err = w.WriteQuad(quad.Quad{
		Subject:   repoIRI,
		Predicate: prdType,
		Object:    typeRepo,
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

// Import imports git repository into graph database
// Returns number of imported commits
func (g *Graph) Import(_ context.Context, gitpath string) (int, error) {
	w := graph.NewWriter(g.store)
	defer w.Close()

	return AsQuads(w, gitpath)
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

func importCommit(w quad.Writer, repoIRI quad.IRI, commit *object.Commit) error {
	commitIRI := quad.IRI("sha1:" + commit.Hash.String())

	if err := w.WriteQuad(quad.Quad{
		Subject:   repoIRI,
		Predicate: prdCommit,
		Object:    commitIRI,
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   commitIRI,
		Predicate: prdType,
		Object:    typeCommit,
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   commitIRI,
		Predicate: prdMetadata,
		Object:    quad.String(commit.String()),
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   commitIRI,
		Predicate: prdMessage,
		Object:    quad.String(commit.Message),
	}); err != nil {
		return err
	}
	if err := importSignature(w, commitIRI, prdAuthor, commit.Author); err != nil {
		return err
	}
	if err := importSignature(w, commitIRI, prdCommiter, commit.Committer); err != nil {
		return err
	}

	// dump parents
	for _, p := range commit.ParentHashes {
		if err := w.WriteQuad(quad.Quad{
			Subject:   commitIRI,
			Predicate: prdParent,
			Object:    quad.IRI("sha1:" + p.String()),
		}); err != nil {
			return err
		}
		if err := w.WriteQuad(quad.Quad{
			Subject:   quad.IRI("sha1:" + p.String()),
			Predicate: prdChild,
			Object:    commitIRI,
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

func importSignature(w quad.Writer, commit, pred quad.IRI, sig object.Signature) error {
	// auto-join authors on exact match
	h := md5.Sum([]byte(sig.Name + "\x00" + sig.Email))
	id := quad.BNode(hex.EncodeToString(h[:]))
	if err := w.WriteQuad(quad.Quad{
		Subject:   commit,
		Predicate: pred,
		Object:    id,
		Label:     quad.Time(sig.When),
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   id,
		Predicate: prdType,
		Object:    typeAuthor,
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   id,
		Predicate: prdName,
		Object:    quad.String(sig.Name),
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   id,
		Predicate: prdEmail,
		Object:    quad.IRI(sig.Email),
	}); err != nil {
		return err
	}
	return nil
}

func importFile(w quad.Writer, commitIRI quad.IRI, file *object.File) error {
	fileIRI := quad.IRI("sha1:" + file.Hash.String())

	if err := w.WriteQuad(quad.Quad{
		Subject:   commitIRI,
		Predicate: prdFile,
		Object:    fileIRI,
		Label:     quad.String(file.Name),
	}); err != nil {
		return err
	}
	if err := w.WriteQuad(quad.Quad{
		Subject:   fileIRI,
		Predicate: prdType,
		Object:    typeFile,
	}); err != nil {
		return err
	}
	return nil
}

func importChange(w quad.Writer, commitIRI quad.IRI, change *object.Change) error {
	action, err := change.Action()
	if err != nil {
		return err
	}

	fromIRI := quad.IRI("sha1:" + change.From.TreeEntry.Hash.String())
	toIRI := quad.IRI("sha1:" + change.To.TreeEntry.Hash.String())

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

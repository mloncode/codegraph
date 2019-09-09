package git

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	bblfsh "github.com/bblfsh/go-client/v4"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/voc/rdf"
	"github.com/cayleygraph/cayley/voc/schema"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

const (
	// TypeRepo node keep (back) references to all to level nodes (so far used only for repos)
	TypeRepo   = quad.IRI("git:Repo")
	TypeBranch = quad.IRI("git:Branch")
	TypeCommit = quad.IRI("git:Commit")
	TypeFile   = quad.IRI("git:File")
	TypeAuthor = quad.IRI("git:Author")

	// node type predicate
	PredType = quad.IRI(rdf.Type)

	// commits
	PredBranch   = quad.IRI("git:branch")
	PredCommit   = quad.IRI("git:commit")
	PredMetadata = quad.IRI("git:metadata")
	PredMessage  = quad.IRI("git:message")
	PredAuthor   = quad.IRI("git:author")
	PredCommiter = quad.IRI("git:commiter")
	PredName     = quad.IRI(schema.Name)
	PredEmail    = quad.IRI("git:email")
	PredChild    = quad.IRI("git:child")
	PredParent   = quad.IRI("git:parent")

	// files
	PredFile     = quad.IRI("git:file")
	PredFilename = quad.IRI("git:filename")
	PredAdd      = quad.IRI("git:add")
	PredRemove   = quad.IRI("git:remove")
	PredModify   = quad.IRI("git:modify")
)

// ExportStats contains Git-to-Quads export statistics.
type ExportStats struct {
	Commits int
	Files   int
	Quads   int
}

// QuadExporter exports one or more Git repositories as quads.
type QuadExporter struct {
	w   quad.Writer
	bw  quad.BatchWriter
	err error
	cli *bblfsh.Client

	Hooks struct {
		OnFile func(id quad.Value, f *object.File) error
	}
	ExportStats
}

// NewQuadExporter creates a new exporter that writes Git objects as quads.
func NewQuadExporter(w quad.Writer) (*QuadExporter, error) {
	exp := &QuadExporter{
		w:  w,
		bw: newBatchWriter(w),
	}
	if err := writeGephiMetadata(w); err != nil {
		return nil, err
	}
	return exp, nil
}

// ExportPath writes a git repository at a given path as a set of quads.
func (e *QuadExporter) ExportPath(gitpath string) error {
	repo, repoIRI, err := openGit(gitpath)
	if err != nil {
		return err
	}

	imp := &repoExporter{
		e:       e,
		repo:    repo,
		repoIRI: repoIRI,
	}
	imp.seen.files = make(map[plumbing.Hash]struct{})
	defer imp.Close()
	return imp.Do()
}

// WriteQuads writes quads to the underlying writer.
func (e *QuadExporter) WriteQuads(q ...quad.Quad) error {
	if e.err != nil {
		return e.err
	}
	var n int
	n, e.err = e.bw.WriteQuads(q)
	e.Quads += n
	return e.err
}

// Close implements io.Closer.
func (e *QuadExporter) Close() error {
	return nil
}

type repoExporter struct {
	e *QuadExporter

	repo    *git.Repository
	repoIRI quad.IRI
	cli     *bblfsh.Client // optional

	seen struct {
		files map[plumbing.Hash]struct{}
	}
}

func (imp *repoExporter) Close() error {
	if imp.cli != nil {
		imp.cli.Close()
	}
	return nil
}

func (imp *repoExporter) Do() error {
	if err := imp.e.w.WriteQuad(quad.Quad{
		Subject:   imp.repoIRI,
		Predicate: PredType,
		Object:    TypeRepo,
	}); err != nil {
		return err
	}
	if err := imp.importBranches(); err != nil {
		return err
	}
	if err := imp.importCommits(); err != nil {
		return err
	}
	return nil
}

func (imp *repoExporter) importBranches() error {
	it, err := imp.repo.Branches()
	if err != nil {
		return err
	}
	defer it.Close()

	return it.ForEach(func(b *plumbing.Reference) error {
		commitIRI := gitHashToIRI(b.Hash())
		branchIRI := imp.repoIRI + "/" + quad.IRI(b.Name())

		return imp.e.WriteQuads([]quad.Quad{
			{
				Subject:   imp.repoIRI,
				Predicate: PredBranch,
				Object:    branchIRI,
			},
			{
				Subject:   branchIRI,
				Predicate: PredCommit,
				Object:    commitIRI,
			},
			{
				Subject:   branchIRI,
				Predicate: PredType,
				Object:    TypeBranch,
			},
			{
				Subject:   branchIRI,
				Predicate: PredName,
				Object:    quad.String(strings.TrimPrefix(string(b.Name()), "refs/heads/")),
			},
		}...)
	})
}

func (imp *repoExporter) importCommits() error {
	it, err := imp.repo.Log(&git.LogOptions{})
	if err != nil {
		return err
	}
	defer it.Close()

	return it.ForEach(func(c *object.Commit) error {
		imp.e.Commits++
		return imp.importCommit(c)
	})
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

func gitHashToIRI(h plumbing.Hash) quad.IRI {
	return quad.IRI("sha1:" + h.String())
}

func (imp *repoExporter) importCommit(commit *object.Commit) error {
	commitIRI := gitHashToIRI(commit.Hash)

	if err := imp.e.WriteQuads([]quad.Quad{
		{
			Subject:   imp.repoIRI,
			Predicate: PredCommit,
			Object:    commitIRI,
		},
		{
			Subject:   commitIRI,
			Predicate: PredType,
			Object:    TypeCommit,
		},
		{
			Subject:   commitIRI,
			Predicate: PredMetadata,
			Object:    quad.String(commit.String()),
		},
		{
			Subject:   commitIRI,
			Predicate: PredMessage,
			Object:    quad.String(commit.Message),
		},
	}...); err != nil {
		return err
	}
	if err := imp.importSignature(commitIRI, PredAuthor, commit.Author); err != nil {
		return err
	}
	if err := imp.importSignature(commitIRI, PredCommiter, commit.Committer); err != nil {
		return err
	}

	// dump parents
	for _, p := range commit.ParentHashes {
		if err := imp.e.WriteQuads([]quad.Quad{
			{
				Subject:   commitIRI,
				Predicate: PredParent,
				Object:    gitHashToIRI(p),
			},
			{
				Subject:   gitHashToIRI(p),
				Predicate: PredChild,
				Object:    commitIRI,
			},
		}...); err != nil {
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
		return imp.importFile(commitIRI, f)
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
		if err = imp.importChange(commitIRI, ch); err != nil {
			return err
		}
	}
	return nil
}

func (imp *repoExporter) importSignature(commit, pred quad.IRI, sig object.Signature) error {
	// auto-join authors on exact match
	h := md5.Sum([]byte(sig.Name + "\x00" + sig.Email))
	id := quad.BNode(hex.EncodeToString(h[:]))

	return imp.e.WriteQuads([]quad.Quad{
		{
			Subject:   commit,
			Predicate: pred,
			Object:    id,
			Label:     quad.Time(sig.When),
		},
		{
			Subject:   id,
			Predicate: PredType,
			Object:    TypeAuthor,
		},
		{
			Subject:   id,
			Predicate: PredName,
			Object:    quad.String(sig.Name),
		},
		{
			Subject:   id,
			Predicate: PredEmail,
			Object:    quad.IRI(sig.Email),
		},
	}...)
}

func (imp *repoExporter) importFile(commitIRI quad.IRI, file *object.File) error {
	fileIRI := gitHashToIRI(file.Hash)

	if err := imp.e.WriteQuads([]quad.Quad{{
		Subject:   commitIRI,
		Predicate: PredFile,
		Object:    fileIRI,
		Label:     quad.String(file.Name),
	}}...); err != nil {
		return err
	}

	if _, ok := imp.seen.files[file.Hash]; ok {
		return nil
	}
	imp.seen.files[file.Hash] = struct{}{}
	imp.e.Files++

	if err := imp.e.WriteQuads([]quad.Quad{
		{
			Subject:   fileIRI,
			Predicate: PredType,
			Object:    TypeFile,
		},
		{
			Subject:   fileIRI,
			Predicate: PredFilename,
			Object:    quad.String(file.Name),
		},
	}...); err != nil {
		return err
	}
	if imp.e.Hooks.OnFile == nil {
		return nil
	}
	return imp.e.Hooks.OnFile(fileIRI, file)
}

func (imp *repoExporter) importChange(commitIRI quad.IRI, change *object.Change) error {
	action, err := change.Action()
	if err != nil {
		return err
	}
	fromIRI := gitHashToIRI(change.From.TreeEntry.Hash)
	toIRI := gitHashToIRI(change.To.TreeEntry.Hash)

	var q quad.Quad
	switch action {
	case merkletrie.Delete:
		q = quad.Quad{
			Subject:   fromIRI,
			Predicate: PredRemove,
			Object:    commitIRI,
			Label:     quad.String(change.From.Name),
		}
	case merkletrie.Insert:
		q = quad.Quad{
			Subject:   toIRI,
			Predicate: PredAdd,
			Object:    commitIRI,
			Label:     quad.String(change.To.Name),
		}
	case merkletrie.Modify:
		q = quad.Quad{
			Subject:   toIRI,
			Predicate: PredModify,
			Object:    commitIRI,
			Label:     quad.String(change.To.Name),
		}
	default:
		return nil
	}
	return imp.e.WriteQuads(q)
}

func newBatchWriter(w quad.Writer) quad.BatchWriter {
	if bw, ok := w.(quad.BatchWriter); ok {
		return &batchWriter{bw: bw}
	}
	return &batchWriter{w: w}
}

type batchWriter struct {
	w  quad.Writer
	bw quad.BatchWriter
}

func (w *batchWriter) WriteQuads(buf []quad.Quad) (int, error) {
	if w.bw != nil {
		return w.bw.WriteQuads(buf)
	}
	for i, q := range buf {
		if err := w.w.WriteQuad(q); err != nil {
			return i, err
		}
	}
	return len(buf), nil
}

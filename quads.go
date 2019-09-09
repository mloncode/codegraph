package codegraph

import (
	"context"
	"io"
	"io/ioutil"
	"strings"
	"time"

	bblfsh "github.com/bblfsh/go-client/v4"
	"github.com/cayleygraph/cayley/quad"
	"github.com/mloncode/codegraph/git"
	"github.com/mloncode/codegraph/uast"
	"github.com/src-d/enry/v2"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Exporter struct {
	w    quad.Writer
	cli  *bblfsh.Client
	opts ExportOptions

	ge *git.QuadExporter
}

type ExportOptions struct {
	UASTs      bool   // export UASTs
	BblfshAddr string // for processing UASTs; defaults to "localhost:9432"
}

func NewExporter(w quad.Writer, opts *ExportOptions) (*Exporter, error) {
	if opts == nil {
		opts = &ExportOptions{}
	}
	if opts.UASTs && opts.BblfshAddr == "" {
		opts.BblfshAddr = "localhost:9432"
	}
	exp := &Exporter{w: w, opts: *opts}
	if opts.UASTs {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		cli, err := bblfsh.NewClientContext(ctx, exp.opts.BblfshAddr)
		if err != nil {
			return nil, err
		}
		exp.cli = cli
	}
	return exp, nil
}

func (e *Exporter) Close() error {
	if e.cli != nil {
		_ = e.cli.Close()
	}
	if e.ge != nil {
		return e.ge.Close()
	}
	return nil
}

func (e *Exporter) ExportRepoPath(gitpath string) error {
	if e.ge == nil {
		ge, err := git.NewQuadExporter(e.w)
		if err != nil {
			return err
		}
		ge.Hooks.OnFile = func(id quad.Value, f *object.File) error {
			rc, err := f.Reader()
			if err != nil {
				return err
			}
			defer rc.Close()
			return e.exportFile(id, f.Name, rc)
		}
		e.ge = ge
	}
	return e.ge.ExportPath(gitpath)
}

func (e *Exporter) exportFile(id quad.Value, name string, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	lang := enry.GetLanguage(name, data)
	if lang != enry.OtherLanguage {
		if err := e.w.WriteQuad(quad.Quad{
			Subject:   id,
			Predicate: predEnryLang,
			Object:    quad.String(lang),
		}); err != nil {
			return err
		}
	}
	if !e.opts.UASTs {
		return nil
	}
	req := e.cli.NewParseRequest().
		Filename(name).
		Content(string(data)).
		Mode(bblfsh.Semantic)
	if lang != enry.OtherLanguage {
		req = req.Language(lang)
	}
	u, _, err := req.UAST()
	if isUnsupportedLanguage(err) || isInvalidEncoding(err) {
		return nil
	} else if err != nil {
		return err
	}
	return uast.AsQuads(e.w, id, u)
}

func isUnsupportedLanguage(err error) bool {
	// TODO(dennwc): return specific error in the client
	return err != nil && strings.Contains(err.Error(), "missing driver for language")
}

func isInvalidEncoding(err error) bool {
	// TODO(dennwc): return specific error in the client
	return err != nil && strings.Contains(err.Error(), "unknown source file encoding")
}

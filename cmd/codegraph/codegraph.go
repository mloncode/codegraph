package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/quad/nquads"
	"github.com/mloncode/codegraph"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func registerOutQuadFlag(f *pflag.FlagSet) *string {
	return f.StringP("out", "o", "-", "write output to a file")
}

func newQuadOutput(out string) (quad.WriteCloser, error) {
	var (
		w io.Writer = os.Stdout
		c []io.Closer
	)
	if out != "" && out != "-" {
		f, err := os.Create(out)
		if err != nil {
			return nil, err
		}
		w = f
		c = append(c, f)
	}
	if strings.HasSuffix(out, ".gz") {
		zw := gzip.NewWriter(w)
		w = zw
		c = append(c, zw)
	}
	qw := nquads.NewWriter(w)
	if len(c) == 0 {
		return qw, nil
	}
	c = append(c, qw)
	return &quadWriteCloser{Writer: qw, c: c}, nil
}

type quadWriteCloser struct {
	quad.Writer
	c []io.Closer
}

func (w *quadWriteCloser) Close() error {
	for i := len(w.c) - 1; i >= 0; i-- {
		if err := w.c[i].Close(); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	cmdQuads := &cobra.Command{
		Use:   "quads <repo> [<repo>...]",
		Short: "Convert Git repository to quads",
	}
	fout := registerOutQuadFlag(cmdQuads.Flags())
	fuast := cmdQuads.Flags().Bool("uast", true, "export UAST of files in Git")
	fbblfsh := cmdQuads.Flags().String("bblfsh", "localhost:9432", "address of Babelfish server for parsing")
	cmdQuads.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("expected at least one argument")
		}
		qw, err := newQuadOutput(*fout)
		if err != nil {
			return err
		}
		exp, err := codegraph.NewExporter(qw, &codegraph.ExportOptions{
			UASTs: *fuast, BblfshAddr: *fbblfsh,
		})
		if err != nil {
			_ = qw.Close()
			return err
		}
		cmd.SilenceUsage = true
		for _, path := range args {
			fmt.Fprintln(os.Stderr, path)
			if err := exp.ExportRepoPath(path); err != nil {
				return err
			}
		}
		if err := exp.Close(); err != nil {
			_ = qw.Close()
			return err
		}
		return qw.Close()
	}
	root.AddCommand(cmdQuads)
}

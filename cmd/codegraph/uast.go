package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bblfsh/sdk/v3/uast/uastyaml"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/quad/nquads"
	"github.com/mloncode/codegraph/uast"
	"github.com/spf13/cobra"
)

func init() {
	cmdUAST := &cobra.Command{
		Use:   "uast <command>",
		Short: "UAST-related commands",
	}
	root.AddCommand(cmdUAST)

	cmdQuads := &cobra.Command{
		Use:   "quads <file> [<files>...]",
		Short: "convert UAST file into quads",
	}
	fout := cmdQuads.Flags().StringP("out", "o", "-", "write output to a file")
	cmdQuads.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("expected at least one argument")
		}
		var (
			w io.Writer = os.Stdout
			c []io.Closer
		)
		if out := *fout; out != "" && out != "-" {
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
			c = append(c, f)
		}
		if strings.HasSuffix(*fout, ".gz") {
			zw := gzip.NewWriter(w)
			w = zw
			c = append(c, zw)
		}
		cmd.SilenceUsage = true
		qw := nquads.NewWriter(w)
		c = append(c, qw)
		for _, path := range args {
			fmt.Fprintln(os.Stderr, path)
			err := func() error {
				var (
					r   io.Reader
					fid quad.Value
				)
				if path == "-" {
					r = os.Stdin
					fid = quad.IRI("stdin")
				} else {
					f, err := os.Open(path)
					if err != nil {
						return err
					}
					defer f.Close()
					r = f
					fid = quad.IRI(filepath.Base(path))
				}
				data, err := ioutil.ReadAll(r)
				if err != nil {
					return err
				}
				ast, err := uastyaml.Unmarshal(data)
				if err != nil {
					return err
				}
				return uast.AsQuads(qw, fid, ast)
			}()
			if err != nil {
				return err
			}
		}
		for i := len(c) - 1; i >= 0; i-- {
			if err := c[i].Close(); err != nil {
				return err
			}
		}
		return nil
	}
	cmdUAST.AddCommand(cmdQuads)
}

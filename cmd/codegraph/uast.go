package main

import (
	"errors"
	"fmt"
	"github.com/bblfsh/sdk/v3/uast/uastyaml"
	"github.com/cayleygraph/cayley/quad"
	"github.com/mloncode/codegraph/uast"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
	fout := registerOutQuadFlag(cmdQuads.Flags())
	cmdQuads.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("expected at least one argument")
		}
		qw, err := newQuadOutput(*fout)
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
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
		return qw.Close()
	}
	cmdUAST.AddCommand(cmdQuads)
}

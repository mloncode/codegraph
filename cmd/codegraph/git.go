package main

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/cayleygraph/cayley/quad/nquads"
	"io"
	"os"
	"strings"

	"github.com/MLonCode/codegraph/git"
	"github.com/spf13/cobra"
)

func init() {
	var g *git.GitGraph
	cmdGit := &cobra.Command{
		Use:   "git <command>",
		Short: "Git-related commands",
	}
	db := cmdGit.PersistentFlags().StringP("db", "a", "./", "database directory")
	cmdGit.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		g, err = git.Open(*db)
		return err
	}
	cmdGit.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if g != nil {
			_ = g.Close()
		}
		return nil
	}
	Root.AddCommand(cmdGit)

	cmdQuads := &cobra.Command{
		Use:   "quads <repo> [<repos>...]",
		Short: "dump a git repository as quads",
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
			_, err := git.AsQuads(qw, path)
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
	cmdGit.AddCommand(cmdQuads)

	cmdExport := &cobra.Command{
		Use:   "export",
		Short: "export the database as quads",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := g.Export(context.TODO(), os.Stdout)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Exported: %d quads\n", n)
			return nil
		},
	}
	cmdGit.AddCommand(cmdExport)

	cmdImport := &cobra.Command{
		Use:   "import <repo> [<repos>...]",
		Short: "import git repositories to the graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("expected at least one argument")
			}
			var total int
			for _, path := range args {
				n, err := g.Import(context.TODO(), path)
				if err != nil {
					return err
				}
				total += n
			}
			fmt.Fprintf(os.Stderr, "Imported: %d commits\n", total)
			return nil
		},
	}
	cmdGit.AddCommand(cmdImport)

	cmdStats := &cobra.Command{
		Use:   "stats",
		Short: "print commit stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("expected at least one argument")
			}
			var total int
			for _, path := range args {
				n, err := g.Import(context.TODO(), path)
				if err != nil {
					return err
				}
				total += n
			}
			fmt.Fprintf(os.Stderr, "Imported: %d commits\n", total)
			return nil
		},
	}
	limit := cmdStats.Flags().IntP("limit", "n", 0, "top commits per git repository (0 means no limit)")
	sort := cmdStats.Flags().String("sort", "touch", "sort commits by [add, remove, modify, touch, file]")
	noMerge := cmdStats.Flags().Bool("nomerge", false, "do not show merge commits")
	cmdStats.RunE = func(cmd *cobra.Command, args []string) error {
		if *limit < 0 {
			*limit = 0
		}

		var by git.SortBy
		switch strings.ToLower(*sort) {
		case "add":
			by = func(cs1, cs2 *git.CommitStats) bool {
				return cs1.NumAdded > cs2.NumAdded
			}

		case "remove":
			by = func(cs1, cs2 *git.CommitStats) bool {
				return cs1.NumRemoved > cs2.NumRemoved
			}

		case "modify":
			by = func(cs1, cs2 *git.CommitStats) bool {
				return cs1.NumModified > cs2.NumModified
			}

		case "file":
			by = func(cs1, cs2 *git.CommitStats) bool {
				return cs1.NumFiles > cs2.NumFiles
			}

		case "touch":
			by = func(cs1, cs2 *git.CommitStats) bool {
				n1 := cs1.NumAdded + cs1.NumRemoved + cs1.NumModified
				n2 := cs2.NumAdded + cs2.NumRemoved + cs2.NumModified
				return n1 > n2
			}
		default:
			return fmt.Errorf("Invalid -sort argument: %v", *sort)
		}

		ctx := context.TODO()
		if err := g.PrintStats(ctx, *limit, by, *noMerge); err != nil {
			return err
		}
		return nil
	}
	cmdGit.AddCommand(cmdStats)
}

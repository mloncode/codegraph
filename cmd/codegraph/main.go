package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var Root = &cobra.Command{
	Use:   "codegraph",
	Short: "tools for working with Git/UAST graphs",
}

func main() {
	if err := Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

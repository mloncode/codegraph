package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var Root = &cobra.Command{
	Use:   "uastgraph",
	Short: "tools for working on UAST graphs",
}

func main() {
	if err := Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

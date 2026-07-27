// Reduit — a local-first Proton Mail CLI with semantic search and MCP.
//
// See https://gitea.stump.rocks/stump.wtf/reduit for documentation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joestump/reduit/internal/cli"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "reduit:", err)
		os.Exit(1)
	}
}

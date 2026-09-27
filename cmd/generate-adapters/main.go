package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/recursivefunctions/d2mcp/internal/distribution"
)

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "fail when generated files differ")
	flag.Parse()

	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	generator, err := distribution.Load(
		filepath.Join(absoluteRoot, "server.json"),
		filepath.Join(absoluteRoot, "distribution", "targets.yaml"),
	)
	if err != nil {
		fail(err)
	}
	if err := generator.Write(absoluteRoot, *check); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/danprince/sietch/internal/errors"
)

func main() {
	var shouldServe bool

	flag.BoolVar(&shouldServe, "serve", false, "Serve & rebuild the site")
	flag.Parse()
	args := flag.Args()

	rootDir, _ := os.Getwd()
	pagesDir := rootDir

	if len(args) == 1 {
		pagesDir = path.Join(rootDir, args[0])
	}

	builder := builderWithDefaults(rootDir)
	builder.pagesDir = pagesDir
	builder.dev = shouldServe

	// Wipe outDir before we build/serve
	os.RemoveAll(builder.outDir)

	if shouldServe {
		Serve(&builder)
		return
	}

	dt, err := builder.build()

	if err != nil {
		fmt.Fprintln(os.Stderr, errors.FmtError(err))
		os.Exit(1)
	} else {
		fmt.Printf("built %d pages in %s", len(builder.pages), dt)
	}
}

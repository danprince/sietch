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

	outDir := path.Join(rootDir, "_site")
	templateFile := path.Join(rootDir, "_template.html")

	builder := builder{
		rootDir:      rootDir,
		pagesDir:     pagesDir,
		outDir:       outDir,
		templateFile: templateFile,
	}

	if shouldServe {
		Serve(&builder)
		return
	}

	err := builder.build()

	if err != nil {
		fmt.Fprintln(os.Stderr, errors.FmtError(err))
		os.Exit(1)
	}
}

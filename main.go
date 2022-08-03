package main

import (
	"flag"
	"os"

	"github.com/danprince/sietch/internal/builder"
)

func main() {
	var shouldServe bool
	flag.BoolVar(&shouldServe, "serve", false, "Serve & rebuild the site")
	flag.Parse()

	rootDir, _ := os.Getwd()

	if shouldServe {
		b := builder.New(rootDir, builder.Development)
		serve(b)
	} else {
		b := builder.New(rootDir, builder.Production)
		b.Build()
	}
}

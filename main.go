package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/danprince/sietch/internal/builder"
)

func main() {
	var shouldServe bool
	flag.BoolVar(&shouldServe, "serve", false, "Serve & rebuild the site")
	flag.Parse()

	rootDir, _ := os.Getwd()

	mode := builder.Production

	if shouldServe {
		mode = builder.Development
	}

	b := builder.New(rootDir, mode)

	// Start from a fresh slate each time the command is run
	os.RemoveAll(b.OutDir)
	os.MkdirAll(b.OutDir, 0755)

	if shouldServe {
		serve(b)
		return
	}

	start := time.Now()
	err := b.Build()
	duration := time.Since(start)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Printf("built site (%s)\n", duration)
	}
}

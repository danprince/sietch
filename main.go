package main

import (
	"os"

	"github.com/danprince/sietch/internal/builder"
)

func main() {
	cwd, _ := os.Getwd()
	b := builder.New(cwd)
	serve(b)
}

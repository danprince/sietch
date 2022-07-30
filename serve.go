package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/danprince/sietch/internal/errors"
)

func Serve(b *builder) {
	var err error

	server := http.FileServer(http.Dir(b.outDir))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rebuild := strings.HasSuffix(r.URL.Path, "/") || strings.HasSuffix(r.URL.Path, ".html")

		// Rebuild the site whenever an HTML file is requested
		if rebuild {
			b.reset()
			dt, err := b.build()

			if err != nil {
				w.Header().Add("Content-type", "text/html")
				w.WriteHeader(500)
				w.Write([]byte(errors.FmtErrorHtml(err)))
				fmt.Fprintln(os.Stderr, errors.FmtError(err))
				return
			}

			fmt.Printf("built %d pages in %s", len(b.pages), dt)
		}

		server.ServeHTTP(w, r)
	}))

	fmt.Println("serving site at http://localhost:8000...")
	err = http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal(err)
	}
}

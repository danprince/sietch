package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/danprince/sietch/internal/builder"
)

func serve(b *builder.Builder) {
	var buildErr error

	server := http.FileServer(http.Dir(b.OutDir))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if buildErr != nil {
			w.Header().Add("Content-type", "text/html")
			w.WriteHeader(500)
			w.Write([]byte(buildErr.Error()))
		} else {
			server.ServeHTTP(w, r)
		}
	}))

	watcher := watch(b.PagesDir, []string{b.OutDir})

	go func() {
		for {
			fmt.Println("building site")
			buildErr = b.Build()
			<-watcher
			b.Reset()
		}
	}()

	fmt.Println("serving site at http://localhost:8000...")
	err := http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal(err)
	}
}

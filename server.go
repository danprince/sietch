package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/danprince/sietch/internal/builder"
	"github.com/danprince/sietch/internal/errors"
	"github.com/danprince/sietch/internal/livereload"
)

func serve(b *builder.Builder) {
	var buildErr error

	lr := livereload.New()
	server := http.FileServer(http.Dir(b.OutDir))

	http.Handle("/ws", lr)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if buildErr != nil {
			w.Header().Add("Content-type", "text/html")
			w.WriteHeader(500)
			w.Write([]byte(errors.Html(buildErr)))
			w.Write([]byte(fmt.Sprintf("<script>%s</script>", livereload.JS)))
		} else {
			server.ServeHTTP(w, r)
		}
	}))

	watcher := watch(b.PagesDir, []string{b.OutDir})

	go func() {
		for {
			fmt.Printf("\x1bc") // clear
			start := time.Now()
			buildErr = b.Build()
			duration := time.Since(start)
			if buildErr != nil {
				fmt.Println(buildErr)
			} else {
				fmt.Printf("built site (%s)\n", duration)
			}
			lr.Notify()
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
